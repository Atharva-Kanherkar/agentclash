package engine

import (
	"encoding/json"
	"testing"

	"github.com/Atharva-Kanherkar/agentclash/backend/internal/sandbox"
)

func TestApplySandboxConfig_WithSandboxBlock(t *testing.T) {
	manifest := json.RawMessage(`{
		"sandbox": {
			"network_access": true,
			"network_allowlist": ["10.0.0.0/8"],
			"env_vars": {"DB_URL": "postgres://localhost"},
			"additional_packages": ["ffmpeg"],
			"sandbox_template_id": "custom-template"
		},
		"version": {
			"number": 1,
			"sandbox_template_id": "pinned-template"
		}
	}`)

	request := &sandbox.CreateRequest{}
	applySandboxConfig(request, manifest)

	if !request.ToolPolicy.AllowNetwork {
		t.Error("expected AllowNetwork=true from sandbox.network_access")
	}
	if len(request.NetworkAllowlist) != 1 || request.NetworkAllowlist[0] != "10.0.0.0/8" {
		t.Errorf("expected network_allowlist=[10.0.0.0/8], got %v", request.NetworkAllowlist)
	}
	if request.EnvVars["DB_URL"] != "postgres://localhost" {
		t.Errorf("expected DB_URL env var, got %v", request.EnvVars)
	}
	if len(request.AdditionalPackages) != 1 || request.AdditionalPackages[0] != "ffmpeg" {
		t.Errorf("expected additional_packages=[ffmpeg], got %v", request.AdditionalPackages)
	}
	// version.sandbox_template_id takes precedence over sandbox.sandbox_template_id
	if request.TemplateID != "pinned-template" {
		t.Errorf("expected TemplateID=pinned-template (from version block), got %q", request.TemplateID)
	}
}

func TestApplySandboxConfig_WithoutSandboxBlock(t *testing.T) {
	manifest := json.RawMessage(`{
		"tool_policy": {"allow_shell": true},
		"version": {"number": 1}
	}`)

	request := &sandbox.CreateRequest{
		ToolPolicy: sandbox.ToolPolicy{AllowNetwork: false},
	}
	applySandboxConfig(request, manifest)

	if request.ToolPolicy.AllowNetwork {
		t.Error("expected AllowNetwork to remain false when no sandbox block")
	}
	if len(request.NetworkAllowlist) != 0 {
		t.Error("expected empty NetworkAllowlist when no sandbox block")
	}
	if len(request.EnvVars) != 0 {
		t.Error("expected empty EnvVars when no sandbox block")
	}
	if len(request.AdditionalPackages) != 0 {
		t.Error("expected empty AdditionalPackages when no sandbox block")
	}
	if request.TemplateID != "" {
		t.Errorf("expected empty TemplateID when no sandbox block, got %q", request.TemplateID)
	}
}

func TestApplySandboxConfig_TemplateIDFromSandboxOnly(t *testing.T) {
	manifest := json.RawMessage(`{
		"sandbox": {
			"sandbox_template_id": "from-sandbox"
		},
		"version": {"number": 1}
	}`)

	request := &sandbox.CreateRequest{}
	applySandboxConfig(request, manifest)

	if request.TemplateID != "from-sandbox" {
		t.Errorf("expected TemplateID=from-sandbox, got %q", request.TemplateID)
	}
}

func TestApplySandboxConfig_InvalidJSON(t *testing.T) {
	request := &sandbox.CreateRequest{}
	applySandboxConfig(request, json.RawMessage(`{invalid`))

	// Should not panic, should leave request unchanged
	if request.TemplateID != "" {
		t.Errorf("expected empty TemplateID on invalid JSON, got %q", request.TemplateID)
	}
}
