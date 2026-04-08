package engine

import (
	"encoding/json"
	"testing"

	"github.com/Atharva-Kanherkar/agentclash/backend/internal/sandbox"
)

func TestBuildToolRegistry_DefaultPrimitivesVisible(t *testing.T) {
	registry, err := buildToolRegistry(sandbox.ToolPolicy{
		AllowedToolKinds: []string{"file"},
		AllowShell:       true,
	}, []byte(`{"challenge":"fixture"}`), []byte(`{}`))
	if err != nil {
		t.Fatalf("buildToolRegistry returned error: %v", err)
	}

	assertRegistryVisibleTools(t, registry, submitToolName, readFileToolName, writeFileToolName, listFilesToolName, execToolName)
}

func TestBuildToolRegistry_AppliesAllowedDeniedAndSnapshotOverridesInOrder(t *testing.T) {
	registry, err := buildToolRegistry(
		sandbox.ToolPolicy{AllowedToolKinds: []string{"file"}, AllowShell: true},
		[]byte(`{
			"tools":{
				"allowed":["read_file","write_file","exec"],
				"denied":["write_file"],
				"custom":[
					{
						"name":"inventory_lookup",
						"description":"Lookup inventory",
						"parameters":{"type":"object","properties":{"sku":{"type":"string"}}},
						"implementation":{"primitive":"exec","args":{"command":["echo","hi"]}}
					}
				]
			}
		}`),
		[]byte(`{"tool_overrides":{"denied":["exec","inventory_lookup"]}}`),
	)
	if err != nil {
		t.Fatalf("buildToolRegistry returned error: %v", err)
	}

	assertRegistryVisibleTools(t, registry, readFileToolName)
	if _, ok := registry.resolveAny(execToolName); !ok {
		t.Fatalf("exec should still be loaded as an internal primitive")
	}
}

func TestBuildToolRegistry_RejectsCustomToolNameCollision(t *testing.T) {
	_, err := buildToolRegistry(
		sandbox.ToolPolicy{AllowedToolKinds: []string{"file"}},
		[]byte(`{
			"tools":{
				"custom":[
					{
						"name":"read_file",
						"description":"bad collision",
						"parameters":{"type":"object"},
						"implementation":{"primitive":"exec","args":{"command":["echo","hi"]}}
					}
				]
			}
		}`),
		nil,
	)
	if err == nil {
		t.Fatal("expected name collision error")
	}
}

func TestRegistryToolDefinitions_OnlyReturnsVisibleTools(t *testing.T) {
	registry, err := buildToolRegistry(
		sandbox.ToolPolicy{AllowedToolKinds: []string{"file"}, AllowShell: true},
		[]byte(`{"tools":{"allowed":["read_file"]}}`),
		nil,
	)
	if err != nil {
		t.Fatalf("buildToolRegistry returned error: %v", err)
	}

	definitions := registry.ToolDefinitions()
	if len(definitions) != 1 {
		t.Fatalf("tool definition count = %d, want 1", len(definitions))
	}
	if definitions[0].Name != readFileToolName {
		t.Fatalf("tool definition = %q, want %q", definitions[0].Name, readFileToolName)
	}
}

func TestDecodeManifestToolsConfig(t *testing.T) {
	config := decodeManifestToolsConfig([]byte(`{
		"tools":{
			"allowed":["read_file","exec"],
			"denied":["exec"],
			"custom":[
				{
					"name":"inventory_lookup",
					"description":"Lookup inventory",
					"parameters":{"type":"object"},
					"implementation":{"primitive":"exec","args":{"command":["echo","hi"]}}
				}
			]
		}
	}`))

	if len(config.Allowed) != 2 || config.Allowed[0] != readFileToolName || config.Allowed[1] != execToolName {
		t.Fatalf("allowed = %#v, want read_file and exec", config.Allowed)
	}
	if len(config.Denied) != 1 || config.Denied[0] != execToolName {
		t.Fatalf("denied = %#v, want exec", config.Denied)
	}
	if len(config.Custom) != 1 || config.Custom[0].Name != "inventory_lookup" {
		t.Fatalf("custom = %#v, want inventory_lookup", config.Custom)
	}
}

func TestDecodeSnapshotToolOverrides_DenyOnly(t *testing.T) {
	overrides := decodeSnapshotToolOverrides([]byte(`{
		"tool_overrides":{
			"denied":[" exec ","read_file"]
		}
	}`))

	if len(overrides.Denied) != 2 || overrides.Denied[0] != execToolName || overrides.Denied[1] != readFileToolName {
		t.Fatalf("denied = %#v, want normalized entries", overrides.Denied)
	}
}

func TestManifestBackedToolDefaultsToStructuredError(t *testing.T) {
	tool, err := newManifestCustomTool(manifestCustomToolConfig{
		Name:           "inventory_lookup",
		Description:    "Lookup inventory",
		Parameters:     json.RawMessage(`{"type":"object"}`),
		Implementation: json.RawMessage(`{"primitive":"exec","args":{"command":["echo","hi"]}}`),
	})
	if err != nil {
		t.Fatalf("newManifestCustomTool returned error: %v", err)
	}

	result, execErr := tool.Execute(t.Context(), ToolExecutionRequest{})
	if execErr != nil {
		t.Fatalf("Execute returned error: %v", execErr)
	}
	if !result.IsError {
		t.Fatalf("expected stub custom tool to return structured error")
	}
}

func assertRegistryVisibleTools(t *testing.T, registry *Registry, want ...string) {
	t.Helper()

	if len(registry.visible) != len(want) {
		t.Fatalf("visible tool count = %d, want %d", len(registry.visible), len(want))
	}
	for _, name := range want {
		if _, ok := registry.Resolve(name); !ok {
			t.Fatalf("tool %q was not visible", name)
		}
	}
}
