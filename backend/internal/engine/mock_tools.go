package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

type MockStrategy string

const (
	MockStrategyStatic MockStrategy = "static"
	MockStrategyLookup MockStrategy = "lookup"
	MockStrategyEcho   MockStrategy = "echo"
)

type mockToolConfig struct {
	Type      string          `json:"type"`
	Strategy  MockStrategy    `json:"strategy"`
	Response  json.RawMessage `json:"response"`
	LookupKey string          `json:"lookup_key"`
	Responses json.RawMessage `json:"responses"`
	Template  json.RawMessage `json:"template"`
}

type mockTool struct {
	name        string
	description string
	parameters  json.RawMessage
	strategy    MockStrategy
	config      mockToolConfig

	// Pre-parsed fields for each strategy.
	staticResponse json.RawMessage            // static
	lookupKey      string                     // lookup
	lookupMap      map[string]json.RawMessage // lookup
	templateMap    map[string]any             // echo
}

func (t *mockTool) Name() string               { return t.name }
func (t *mockTool) Description() string         { return t.description }
func (t *mockTool) Parameters() json.RawMessage { return cloneJSON(t.parameters) }
func (t *mockTool) Category() ToolCategory      { return ToolCategoryMock }

func (t *mockTool) Execute(_ context.Context, request ToolExecutionRequest) (ToolExecutionResult, error) {
	switch t.strategy {
	case MockStrategyStatic:
		return t.executeStatic()
	case MockStrategyLookup:
		return t.executeLookup(request.Args)
	case MockStrategyEcho:
		return t.executeEcho(request.Args)
	default:
		return ToolExecutionResult{
			Content: encodeToolErrorMessage(fmt.Sprintf("unknown mock strategy %q", t.strategy)),
			IsError: true,
		}, nil
	}
}

func (t *mockTool) executeStatic() (ToolExecutionResult, error) {
	return ToolExecutionResult{Content: string(t.staticResponse)}, nil
}

func (t *mockTool) executeLookup(args json.RawMessage) (ToolExecutionResult, error) {
	var parsed map[string]any
	if len(args) > 0 {
		if err := json.Unmarshal(args, &parsed); err != nil {
			return ToolExecutionResult{
				Content: encodeToolErrorMessage("failed to decode arguments"),
				IsError: true,
			}, nil
		}
	}

	keyValue := resolveKeyPath(parsed, t.lookupKey)

	if response, ok := t.lookupMap[keyValue]; ok {
		return ToolExecutionResult{Content: string(response)}, nil
	}
	if response, ok := t.lookupMap["*"]; ok {
		return ToolExecutionResult{Content: string(response)}, nil
	}

	return ToolExecutionResult{
		Content: encodeToolErrorMessage(fmt.Sprintf("no mock response for %s=%q and no fallback defined", t.lookupKey, keyValue)),
		IsError: true,
	}, nil
}

func (t *mockTool) executeEcho(args json.RawMessage) (ToolExecutionResult, error) {
	var parsed map[string]any
	if len(args) > 0 {
		if err := json.Unmarshal(args, &parsed); err != nil {
			return ToolExecutionResult{
				Content: encodeToolErrorMessage("failed to decode arguments"),
				IsError: true,
			}, nil
		}
	}
	if parsed == nil {
		parsed = map[string]any{}
	}

	resolved := substituteTemplate(t.templateMap, parsed)

	encoded, err := json.Marshal(resolved)
	if err != nil {
		return ToolExecutionResult{
			Content: encodeToolErrorMessage("failed to encode echo response"),
			IsError: true,
		}, nil
	}
	return ToolExecutionResult{Content: string(encoded)}, nil
}

func substituteTemplate(template map[string]any, args map[string]any) map[string]any {
	result := make(map[string]any, len(template))
	for key, value := range template {
		result[key] = substituteValue(value, args)
	}
	return result
}

func substituteValue(value any, args map[string]any) any {
	switch v := value.(type) {
	case string:
		return substituteString(v, args)
	case map[string]any:
		return substituteTemplate(v, args)
	case []any:
		out := make([]any, len(v))
		for i, elem := range v {
			out[i] = substituteValue(elem, args)
		}
		return out
	default:
		return value
	}
}

func substituteString(s string, args map[string]any) string {
	result := s
	for paramName, paramValue := range args {
		placeholder := "${" + paramName + "}"
		if !strings.Contains(result, placeholder) {
			continue
		}
		var replacement string
		switch v := paramValue.(type) {
		case string:
			replacement = v
		default:
			encoded, err := json.Marshal(v)
			if err != nil {
				replacement = fmt.Sprintf("%v", v)
			} else {
				replacement = string(encoded)
			}
		}
		result = strings.ReplaceAll(result, placeholder, replacement)
	}
	return result
}

func resolveKeyPath(obj map[string]any, keyPath string) string {
	segments := strings.Split(keyPath, ".")
	var current any = obj
	for _, seg := range segments {
		m, ok := current.(map[string]any)
		if !ok {
			return ""
		}
		current, ok = m[seg]
		if !ok {
			return ""
		}
	}
	switch v := current.(type) {
	case string:
		return v
	default:
		encoded, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", v)
		}
		return strings.TrimSpace(string(encoded))
	}
}

func validateTemplatePlaceholders(value any, path string) error {
	switch v := value.(type) {
	case string:
		return validatePlaceholderSyntax(v, path)
	case map[string]any:
		for key, child := range v {
			childPath := path + "." + key
			if err := validateTemplatePlaceholders(child, childPath); err != nil {
				return err
			}
		}
	case []any:
		for i, child := range v {
			childPath := fmt.Sprintf("%s[%d]", path, i)
			if err := validateTemplatePlaceholders(child, childPath); err != nil {
				return err
			}
		}
	}
	return nil
}

func validatePlaceholderSyntax(s string, path string) error {
	rest := s
	for {
		idx := strings.Index(rest, "${")
		if idx == -1 {
			return nil
		}
		after := rest[idx+2:]
		closeIdx := strings.Index(after, "}")
		if closeIdx == -1 {
			return fmt.Errorf("unclosed placeholder at %s: %q", path, rest[idx:])
		}
		varName := after[:closeIdx]
		if strings.TrimSpace(varName) == "" {
			return fmt.Errorf("empty placeholder at %s: %q", path, rest[idx:idx+2+closeIdx+1])
		}
		rest = after[closeIdx+1:]
	}
}

func newMockTool(name string, description string, parameters json.RawMessage, config mockToolConfig) (*mockTool, error) {
	strategy := MockStrategy(strings.TrimSpace(strings.ToLower(string(config.Strategy))))

	// Infer strategy if not explicitly set.
	if strategy == "" {
		switch {
		case len(config.LookupKey) > 0 || len(config.Responses) > 0:
			strategy = MockStrategyLookup
		case len(config.Template) > 0:
			strategy = MockStrategyEcho
		default:
			strategy = MockStrategyStatic
		}
	}

	tool := &mockTool{
		name:        name,
		description: description,
		parameters:  parameters,
		strategy:    strategy,
		config:      config,
	}

	switch strategy {
	case MockStrategyStatic:
		if len(config.Response) == 0 {
			return nil, fmt.Errorf("mock tool %q with static strategy requires a response field", name)
		}
		if !json.Valid(config.Response) {
			return nil, fmt.Errorf("mock tool %q static response is not valid JSON", name)
		}
		tool.staticResponse = cloneJSON(config.Response)

	case MockStrategyLookup:
		key := strings.TrimSpace(config.LookupKey)
		if key == "" {
			return nil, fmt.Errorf("mock tool %q with lookup strategy requires a lookup_key field", name)
		}
		if len(config.Responses) == 0 {
			return nil, fmt.Errorf("mock tool %q with lookup strategy requires a responses field", name)
		}
		var responsesMap map[string]json.RawMessage
		if err := json.Unmarshal(config.Responses, &responsesMap); err != nil {
			return nil, fmt.Errorf("mock tool %q lookup responses must be a JSON object: %w", name, err)
		}
		for k, v := range responsesMap {
			if !json.Valid(v) {
				return nil, fmt.Errorf("mock tool %q lookup response for key %q is not valid JSON", name, k)
			}
		}
		tool.lookupKey = key
		tool.lookupMap = responsesMap

	case MockStrategyEcho:
		if len(config.Template) == 0 {
			return nil, fmt.Errorf("mock tool %q with echo strategy requires a template field", name)
		}
		var templateMap map[string]any
		if err := json.Unmarshal(config.Template, &templateMap); err != nil {
			return nil, fmt.Errorf("mock tool %q echo template must be a JSON object: %w", name, err)
		}
		if err := validateTemplatePlaceholders(templateMap, "template"); err != nil {
			return nil, fmt.Errorf("mock tool %q has invalid echo template: %w", name, err)
		}
		tool.templateMap = templateMap

	default:
		return nil, fmt.Errorf("mock tool %q has unknown strategy %q; supported: static, lookup, echo", name, strategy)
	}

	return tool, nil
}
