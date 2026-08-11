package api

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/agentclash/agentclash/backend/internal/budget"
	"github.com/agentclash/agentclash/backend/internal/repository"
	"github.com/agentclash/agentclash/runtime/challengepack"
	"github.com/agentclash/agentclash/runtime/provider"
	"github.com/google/uuid"
)

// Generating a challenge-pack draft from a plain-English description of an app.
//
// This is a single-shot structured-output call, not an agent loop: the whole
// output is one typed artifact (a pack composition), the visual builder is the
// iteration surface, and publish is the human gate. So there is no transcript,
// no tool registry, and nothing new to persist — the draft is the state, and it
// is created through the builder's own CreateDraft so authorization, execution
// -mode validation, and the compile/publish path stay in one place.

// challengePackGenerateTimeout bounds the single provider call. Authoring a
// pack is a larger completion than an insight summary, so it gets more room.
const challengePackGenerateTimeout = 90 * time.Second

// challengePackGenerateFeature names this feature in provider call metadata and
// spend logs, and is its per-workspace rate-limit bucket (see
// ratelimit.limiterForGroup).
const challengePackGenerateFeature = "challenge_pack_generate"

// ChallengePackGenerationRepository is the data access draft generation needs
// on top of the builder's own: the BYOK provider account to call, the secrets
// that back its credential reference, and the workspace's spend policies.
type ChallengePackGenerationRepository interface {
	GetProviderAccountByID(ctx context.Context, id uuid.UUID) (repository.ProviderAccountRow, error)
	LoadWorkspaceSecrets(ctx context.Context, workspaceID uuid.UUID) (map[string]string, error)
	ListSpendPoliciesByWorkspaceID(ctx context.Context, workspaceID uuid.UUID) ([]repository.SpendPolicyRow, error)
}

// ChallengePackGenerationConfig wires the optional generation dependencies.
// Client and Repo are required; without them the manager reports generation
// unavailable rather than half-working.
type ChallengePackGenerationConfig struct {
	Client        provider.Client
	Repo          ChallengePackGenerationRepository
	BudgetChecker budget.BudgetChecker
	RateLimiter   WorkspaceRateLimiter
	Timeout       time.Duration
}

type challengePackGeneration struct {
	client        provider.Client
	repo          ChallengePackGenerationRepository
	budgetChecker budget.BudgetChecker
	limiter       WorkspaceRateLimiter
	timeout       time.Duration
}

// GenerateChallengePackDraftInput is one description plus the workspace
// provider account and model that should author it (BYOK, exactly like ranking
// insights — there is no server-side generation key).
type GenerateChallengePackDraftInput struct {
	WorkspaceID       uuid.UUID
	Description       string
	ProviderAccountID uuid.UUID
	Model             string
	CreatedBy         uuid.UUID
}

// ChallengePackGenerationValidationError is a caller-fixable problem: a blank
// description, an unusable provider account, or a model reply that does not
// compile into a pack.
type ChallengePackGenerationValidationError struct {
	Code    string
	Message string
}

func (e ChallengePackGenerationValidationError) Error() string { return e.Message }

type ChallengePackGenerationRateLimitError struct {
	RetryAfter time.Duration
}

func (e ChallengePackGenerationRateLimitError) Error() string {
	return "challenge pack generation rate limited"
}

var errChallengePackGenerationUnavailable = errors.New("challenge pack draft generation is not configured")

// WithDraftGeneration attaches the provider client and data access that
// description-to-draft generation needs. Without it, the generate endpoint
// reports itself unavailable instead of failing mid-request.
func (m *ChallengePackBuilderManager) WithDraftGeneration(cfg ChallengePackGenerationConfig) *ChallengePackBuilderManager {
	if cfg.Client == nil || cfg.Repo == nil {
		return m
	}
	checker := cfg.BudgetChecker
	if checker == nil {
		checker = budget.NoopChecker{}
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = challengePackGenerateTimeout
	}
	m.generation = &challengePackGeneration{
		client:        cfg.Client,
		repo:          cfg.Repo,
		budgetChecker: checker,
		limiter:       cfg.RateLimiter,
		timeout:       timeout,
	}
	return m
}

// GenerateDraft turns a description of an app into an editable challenge-pack
// draft: one task, its inputs, deterministic validators, and any judges the
// description justifies. The generated composition is validated through the
// builder's own compile path before anything is written, so generation can
// fail but it can never leave an uncompilable draft behind.
func (m *ChallengePackBuilderManager) GenerateDraft(ctx context.Context, caller Caller, input GenerateChallengePackDraftInput) (repository.ChallengePackDraft, error) {
	generation := m.generation
	if generation == nil {
		return repository.ChallengePackDraft{}, errChallengePackGenerationUnavailable
	}
	// Authorize before spending: generation costs the workspace real provider
	// tokens, and CreateDraft would reject the same caller at the end anyway.
	if err := m.authorize(ctx, caller, input.WorkspaceID); err != nil {
		return repository.ChallengePackDraft{}, err
	}

	description := strings.TrimSpace(input.Description)
	switch {
	case description == "":
		return repository.ChallengePackDraft{}, ChallengePackGenerationValidationError{
			Code:    "invalid_description",
			Message: "description is required",
		}
	case len(description) > maxGeneratedPackDescriptionBytes:
		return repository.ChallengePackDraft{}, ChallengePackGenerationValidationError{
			Code:    "invalid_description",
			Message: fmt.Sprintf("description must be at most %d characters", maxGeneratedPackDescriptionBytes),
		}
	}
	model := strings.TrimSpace(input.Model)
	if model == "" {
		return repository.ChallengePackDraft{}, ChallengePackGenerationValidationError{
			Code:    "invalid_model",
			Message: "model is required",
		}
	}

	allowed, err := workspaceSpendAllowed(ctx, generation.repo, generation.budgetChecker, input.WorkspaceID, challengePackGenerateFeature)
	if err != nil {
		return repository.ChallengePackDraft{}, err
	}
	if !allowed {
		return repository.ChallengePackDraft{}, ChallengePackGenerationValidationError{
			Code:    "budget_exceeded",
			Message: "workspace spend limit exceeded for pack generation",
		}
	}
	if generation.limiter != nil {
		if ok, retryAfter := generation.limiter.Allow(input.WorkspaceID, challengePackGenerateFeature); !ok {
			return repository.ChallengePackDraft{}, ChallengePackGenerationRateLimitError{RetryAfter: retryAfter}
		}
	}

	providerAccount, err := generation.repo.GetProviderAccountByID(ctx, input.ProviderAccountID)
	if err != nil {
		return repository.ChallengePackDraft{}, err
	}
	if !providerAccountVisibleToWorkspace(providerAccount, input.WorkspaceID) {
		return repository.ChallengePackDraft{}, ChallengePackGenerationValidationError{
			Code:    "invalid_provider_account_id",
			Message: "provider_account_id must reference an active provider account visible to this workspace",
		}
	}

	invokeCtx, err := provider.PrepareCredentialContext(ctx, providerAccount.CredentialReference, func() (map[string]string, error) {
		return generation.repo.LoadWorkspaceSecrets(ctx, input.WorkspaceID)
	})
	if err != nil {
		return repository.ChallengePackDraft{}, err
	}

	prompt, err := buildGeneratePackPrompt(description)
	if err != nil {
		return repository.ChallengePackDraft{}, fmt.Errorf("build pack generation prompt: %w", err)
	}

	response, err := generation.client.InvokeModel(invokeCtx, provider.Request{
		ProviderKey:         providerAccount.ProviderKey,
		ProviderAccountID:   providerAccount.ID.String(),
		CredentialReference: providerAccount.CredentialReference,
		Model:               model,
		StepTimeout:         generation.timeout,
		Messages: []provider.Message{
			{Role: "system", Content: generatedPackSystemPrompt},
			{Role: "user", Content: prompt},
		},
		Metadata: mustMarshalJSON(map[string]any{
			"workspace_id":        input.WorkspaceID,
			"provider_account_id": providerAccount.ID,
			"model":               model,
			"feature":             challengePackGenerateFeature,
		}),
	})
	if err != nil {
		return repository.ChallengePackDraft{}, err
	}

	bundle, err := generatedPackDraftBundle(response.OutputText, model, uuid.New())
	if err != nil {
		return repository.ChallengePackDraft{}, err
	}
	composition, err := packBundleComposition(bundle)
	if err != nil {
		return repository.ChallengePackDraft{}, err
	}

	// CreateDraft re-authorizes for publish_challenge_pack and owns draft
	// validation; generation never writes to the repository itself.
	return m.CreateDraft(ctx, caller, CreateDraftInput{
		WorkspaceID:   input.WorkspaceID,
		Name:          bundle.Pack.Name,
		ExecutionMode: challengepack.ExecutionModeNative,
		Composition:   composition,
		CreatedBy:     input.CreatedBy,
	})
}

// generatedPackDraftBundle parses the model's reply and returns the bundle to
// persist, or a validation error naming what was wrong with it.
//
// A bundle that only fails because of its judges is rebuilt without them (no
// second provider call, just a second assembly of the same reply) — the
// same degradation the tryout mapper applies: judge declarations are the part
// most likely to be subtly illegal (a model id that maps to no judge provider,
// a rubric that came back empty), and a deterministic pack the author can open
// and add judges to beats a hard failure. Anything else is reported.
func generatedPackDraftBundle(raw, judgeModel string, sourceID uuid.UUID) (challengepack.Bundle, error) {
	blueprint, err := parseGeneratedPackBlueprint(raw)
	if err != nil {
		return challengepack.Bundle{}, ChallengePackGenerationValidationError{
			Code:    "invalid_generated_pack",
			Message: fmt.Sprintf("pack generation model returned invalid output: %v", err),
		}
	}

	bundle := generatedPackBundle(blueprint, judgeModel, sourceID, true)
	compileErr := packBundleCompileError(bundle)
	if compileErr == nil {
		return bundle, nil
	}
	if len(blueprint.Judges) > 0 {
		if withoutJudges := generatedPackBundle(blueprint, judgeModel, sourceID, false); packBundlePublishable(withoutJudges) {
			slog.Default().Warn("dropped judges from generated challenge pack draft",
				"pack_slug", bundle.Pack.Slug,
				"error", compileErr,
			)
			return withoutJudges, nil
		}
	}
	return challengepack.Bundle{}, ChallengePackGenerationValidationError{
		Code:    "invalid_generated_pack",
		Message: fmt.Sprintf("pack generation model returned a pack that does not compile: %v", compileErr),
	}
}

// --- HTTP handler ---

type generateChallengePackDraftRequest struct {
	Description       string `json:"description"`
	ProviderAccountID string `json:"provider_account_id"`
	Model             string `json:"model"`
}

func generateChallengePackDraftHandler(logger *slog.Logger, service ChallengePackBuilderService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		caller, workspaceID, ok := builderRequestContext(w, r)
		if !ok {
			return
		}
		if err := requireJSONContentType(r); err != nil {
			writeError(w, http.StatusUnsupportedMediaType, "unsupported_media_type", err.Error())
			return
		}
		var req generateChallengePackDraftRequest
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "request body must be valid JSON")
			return
		}
		providerAccountID, err := uuid.Parse(strings.TrimSpace(req.ProviderAccountID))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_provider_account_id", "provider_account_id must be a valid UUID")
			return
		}

		draft, err := service.GenerateDraft(r.Context(), caller, GenerateChallengePackDraftInput{
			WorkspaceID:       workspaceID,
			Description:       req.Description,
			ProviderAccountID: providerAccountID,
			Model:             req.Model,
			CreatedBy:         caller.UserID,
		})
		if err != nil {
			writeChallengePackGenerationError(w, logger, err)
			return
		}
		writeJSON(w, http.StatusCreated, buildChallengePackDraftResponse(draft))
	}
}

// writeChallengePackGenerationError maps the failures unique to generation, and
// defers everything a draft write can fail with to the builder's own mapper.
func writeChallengePackGenerationError(w http.ResponseWriter, logger *slog.Logger, err error) {
	var validationErr ChallengePackGenerationValidationError
	var rateLimitErr ChallengePackGenerationRateLimitError
	var providerFailure provider.Failure
	switch {
	case errors.Is(err, errChallengePackGenerationUnavailable):
		writeError(w, http.StatusServiceUnavailable, "service_unavailable", "challenge pack generation is not configured")
	case errors.As(err, &validationErr):
		writeError(w, http.StatusBadRequest, validationErr.Code, validationErr.Message)
	case errors.As(err, &rateLimitErr):
		writeRetryAfterError(w, http.StatusTooManyRequests, "pack_generation_rate_limited", "too many pack generations for this workspace; retry later", rateLimitErr.RetryAfter)
	case errors.Is(err, repository.ErrProviderAccountNotFound):
		writeError(w, http.StatusBadRequest, "invalid_provider_account_id", "provider_account_id must reference an active provider account visible to this workspace")
	case errors.As(err, &providerFailure):
		writeProviderFailure(w, "pack_generation", "pack generation", providerFailure)
	default:
		handleChallengePackBuilderError(w, logger, err)
	}
}
