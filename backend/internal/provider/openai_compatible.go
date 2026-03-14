package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
)

const defaultOpenAIBaseURL = "https://api.openai.com/v1"

type OpenAICompatibleClient struct {
	httpClient         *http.Client
	baseURL            string
	credentialResolver CredentialResolver
}

func NewOpenAICompatibleClient(httpClient *http.Client, baseURL string, credentialResolver CredentialResolver) OpenAICompatibleClient {
	if httpClient == nil {
		httpClient = &http.Client{}
	}
	if strings.TrimSpace(baseURL) == "" {
		baseURL = defaultOpenAIBaseURL
	}
	return OpenAICompatibleClient{
		httpClient:         httpClient,
		baseURL:            strings.TrimRight(baseURL, "/"),
		credentialResolver: credentialResolver,
	}
}

func (c OpenAICompatibleClient) InvokeModel(ctx context.Context, request Request) (Response, error) {
	apiKey, err := c.credentialResolver.Resolve(ctx, request.CredentialReference)
	if err != nil {
		return Response{}, normalizeCredentialError(request.ProviderKey, err)
	}

	body := struct {
		Model    string          `json:"model"`
		Messages []Message       `json:"messages"`
		Metadata json.RawMessage `json:"metadata,omitempty"`
	}{
		Model:    request.Model,
		Messages: request.Messages,
		Metadata: request.Metadata,
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return Response{}, NewFailure(request.ProviderKey, FailureCodeInvalidRequest, "marshal provider request", false, err)
	}

	callCtx := ctx
	if request.StepTimeout > 0 {
		var cancel context.CancelFunc
		callCtx, cancel = context.WithTimeout(ctx, request.StepTimeout)
		defer cancel()
	}

	req, err := http.NewRequestWithContext(callCtx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return Response{}, NewFailure(request.ProviderKey, FailureCodeInvalidRequest, "build provider request", false, err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return Response{}, classifyTransportError(request.ProviderKey, err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return Response{}, NewFailure(request.ProviderKey, FailureCodeUnavailable, "read provider response", true, err)
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return Response{}, normalizeOpenAIErrorResponse(request.ProviderKey, resp.StatusCode, raw)
	}

	var completion openAICompletionResponse
	if err := json.Unmarshal(raw, &completion); err != nil {
		return Response{}, NewFailure(request.ProviderKey, FailureCodeMalformedResponse, "decode provider response", false, err)
	}
	if len(completion.Choices) != 1 {
		return Response{}, NewFailure(request.ProviderKey, FailureCodeMalformedResponse, "provider response must contain exactly one choice", false, nil)
	}

	return Response{
		ProviderKey:     request.ProviderKey,
		ProviderModelID: completion.Model,
		FinishReason:    completion.Choices[0].FinishReason,
		OutputText:      completion.Choices[0].Message.Content,
		Usage: Usage{
			InputTokens:  completion.Usage.PromptTokens,
			OutputTokens: completion.Usage.CompletionTokens,
			TotalTokens:  completion.Usage.TotalTokens,
		},
		RawResponse: append([]byte(nil), raw...),
	}, nil
}

type openAICompletionResponse struct {
	Model   string `json:"model"`
	Choices []struct {
		FinishReason string `json:"finish_reason"`
		Message      struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int64 `json:"prompt_tokens"`
		CompletionTokens int64 `json:"completion_tokens"`
		TotalTokens      int64 `json:"total_tokens"`
	} `json:"usage"`
}

type openAIErrorEnvelope struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}

func normalizeCredentialError(providerKey string, err error) error {
	if failure, ok := AsFailure(err); ok {
		failure.ProviderKey = providerKey
		return failure
	}
	return NewFailure(providerKey, FailureCodeCredentialUnavailable, err.Error(), false, err)
}

func classifyTransportError(providerKey string, err error) error {
	if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		return NewFailure(providerKey, FailureCodeTimeout, "provider request timed out", true, err)
	}
	if strings.Contains(strings.ToLower(err.Error()), "context deadline exceeded") {
		return NewFailure(providerKey, FailureCodeTimeout, "provider request timed out", true, err)
	}
	return NewFailure(providerKey, FailureCodeUnavailable, "provider request failed", true, err)
}

func normalizeOpenAIErrorResponse(providerKey string, statusCode int, raw []byte) error {
	var envelope openAIErrorEnvelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return NewFailure(
			providerKey,
			FailureCodeMalformedResponse,
			fmt.Sprintf("provider returned HTTP %d with invalid error payload", statusCode),
			false,
			err,
		)
	}

	message := envelope.Error.Message
	if strings.TrimSpace(message) == "" {
		message = fmt.Sprintf("provider returned HTTP %d", statusCode)
	}

	switch statusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		return NewFailure(providerKey, FailureCodeAuth, message, false, nil)
	case http.StatusTooManyRequests:
		return NewFailure(providerKey, FailureCodeRateLimit, message, true, nil)
	case http.StatusBadRequest, http.StatusUnprocessableEntity:
		return NewFailure(providerKey, FailureCodeInvalidRequest, message, false, nil)
	case http.StatusGatewayTimeout, http.StatusBadGateway, http.StatusServiceUnavailable:
		return NewFailure(providerKey, FailureCodeUnavailable, message, true, nil)
	default:
		return NewFailure(providerKey, FailureCodeUnknown, message, statusCode >= 500, nil)
	}
}
