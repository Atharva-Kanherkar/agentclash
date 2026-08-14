package connection

import (
	"context"
	"errors"
	"net/netip"
	"strings"
	"testing"

	"github.com/agentclash/agentclash/backend/internal/repository"
	"github.com/agentclash/agentclash/runtime/provider"
	"github.com/google/uuid"
)

type endpointResolverFunc func(context.Context, string, string) ([]netip.Addr, error)

func (f endpointResolverFunc) LookupNetIP(ctx context.Context, network, host string) ([]netip.Addr, error) {
	return f(ctx, network, host)
}

// fakeRepo implements connection.Repository.
type fakeRepo struct {
	orgID         uuid.UUID
	orgCalls      int
	secrets       map[string]string
	created       repository.CreateProviderAccountParams
	createCalls   int
	upsertedKey   string
	upsertedValue string
}

func (r *fakeRepo) GetOrganizationIDByWorkspaceID(context.Context, uuid.UUID) (uuid.UUID, error) {
	r.orgCalls++
	if r.orgID != uuid.Nil {
		return r.orgID, nil
	}
	return uuid.New(), nil
}
func (r *fakeRepo) CreateProviderAccount(_ context.Context, p repository.CreateProviderAccountParams) (repository.ProviderAccountRow, error) {
	r.createCalls++
	r.created = p
	return repository.ProviderAccountRow{ID: uuid.New(), ProviderKey: p.ProviderKey, Name: p.Name, CredentialReference: p.CredentialReference, BaseURL: p.BaseURL}, nil
}
func (r *fakeRepo) GetProviderAccountByID(context.Context, uuid.UUID) (repository.ProviderAccountRow, error) {
	return repository.ProviderAccountRow{}, repository.ErrProviderAccountNotFound
}
func (r *fakeRepo) ListProviderAccountsByWorkspaceID(context.Context, uuid.UUID) ([]repository.ProviderAccountRow, error) {
	return nil, nil
}
func (r *fakeRepo) ArchiveProviderAccount(context.Context, uuid.UUID) error { return nil }
func (r *fakeRepo) UpsertWorkspaceSecret(_ context.Context, p repository.UpsertWorkspaceSecretParams) error {
	r.upsertedKey = p.Key
	r.upsertedValue = p.Value
	return nil
}
func (r *fakeRepo) LoadWorkspaceSecrets(context.Context, uuid.UUID) (map[string]string, error) {
	return r.secrets, nil
}

// fakeRouter implements connection.ProviderRouter.
type fakeRouter struct {
	invokeResp  provider.Response
	invokeErr   error
	invokeCalls int
	lastInvoke  provider.Request
	models      []provider.ModelInfo
	modelsErr   error
	listCalls   int
	lastList    provider.ListModelsRequest
}

func (r *fakeRouter) InvokeModel(_ context.Context, req provider.Request) (provider.Response, error) {
	r.invokeCalls++
	r.lastInvoke = req
	return r.invokeResp, r.invokeErr
}
func (r *fakeRouter) ListModels(_ context.Context, req provider.ListModelsRequest) ([]provider.ModelInfo, error) {
	r.listCalls++
	r.lastList = req
	return r.models, r.modelsErr
}

func TestCreateStoresAPIKeyAsWorkspaceSecret(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo, &fakeRouter{})

	row, err := svc.Create(context.Background(), uuid.New(), CreateConnectionInput{
		ProviderKey: "openai",
		Name:        "OpenAI Prod",
		APIKey:      "sk-secret",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if repo.upsertedKey != "PROVIDER_OPENAI_API_KEY" || repo.upsertedValue != "sk-secret" {
		t.Fatalf("secret upsert = %q/%q", repo.upsertedKey, repo.upsertedValue)
	}
	if row.CredentialReference != "workspace-secret://PROVIDER_OPENAI_API_KEY" {
		t.Fatalf("credential reference = %q", row.CredentialReference)
	}
}

func TestCreateCanonicalizesAndValidatesCustomEndpoint(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo, &fakeRouter{})
	validatedRaw := ""
	svc.validateBaseURL = func(_ context.Context, raw string) (string, error) {
		validatedRaw = raw
		return "https://models.example.com/openai/v1", nil
	}

	row, err := svc.Create(context.Background(), uuid.New(), CreateConnectionInput{
		ProviderKey: " CuStOm ",
		Name:        "Compatible",
		APIKey:      "custom-secret",
		BaseURL:     "  https://MODELS.example.com/openai/v1/ ",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if validatedRaw != "https://MODELS.example.com/openai/v1/" {
		t.Fatalf("validated raw URL = %q", validatedRaw)
	}
	if row.ProviderKey != "custom" || row.BaseURL != "https://models.example.com/openai/v1" {
		t.Fatalf("created row = %#v", row)
	}
	if repo.created.ProviderKey != "custom" || repo.created.BaseURL != row.BaseURL {
		t.Fatalf("create params = %#v", repo.created)
	}
	if repo.upsertedKey != "PROVIDER_CUSTOM_API_KEY" {
		t.Fatalf("secret key = %q", repo.upsertedKey)
	}
}

func TestCreateRejectsEndpointBeforeAnyPersistenceWrite(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo, &fakeRouter{})
	svc.validateBaseURL = func(context.Context, string) (string, error) {
		return "", func() error {
			_, err := provider.NormalizeBaseURL("http://10.0.0.1/v1")
			return err
		}()
	}

	_, err := svc.Create(context.Background(), uuid.New(), CreateConnectionInput{
		ProviderKey: "custom",
		Name:        "Unsafe",
		APIKey:      "must-not-be-stored",
		BaseURL:     "https://unsafe.example/v1",
	})
	if !provider.IsEndpointValidationError(err) {
		t.Fatalf("Create error = %v, want endpoint validation error", err)
	}
	if repo.orgCalls != 0 || repo.createCalls != 0 || repo.upsertedKey != "" {
		t.Fatalf("persistence touched after validation failure: %#v", repo)
	}
}

func TestCreateRejectsMixedDNSAnswersBeforeAnyPersistenceWrite(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo, &fakeRouter{})
	resolver := endpointResolverFunc(func(context.Context, string, string) ([]netip.Addr, error) {
		return []netip.Addr{
			netip.MustParseAddr("93.184.216.34"),
			netip.MustParseAddr("169.254.169.254"),
		}, nil
	})
	svc.validateBaseURL = func(ctx context.Context, raw string) (string, error) {
		return provider.ValidateBaseURLWithResolver(ctx, raw, resolver)
	}

	_, err := svc.Create(context.Background(), uuid.New(), CreateConnectionInput{
		ProviderKey: "custom",
		Name:        "Mixed DNS",
		APIKey:      "must-not-be-stored",
		BaseURL:     "https://models.example.net/v1",
	})
	if !provider.IsEndpointValidationError(err) {
		t.Fatalf("Create error = %v, want endpoint validation error", err)
	}
	if repo.orgCalls != 0 || repo.createCalls != 0 || repo.upsertedKey != "" {
		t.Fatalf("persistence touched after mixed DNS answer: %#v", repo)
	}
}

func TestCreateRequiresCustomEndpointBeforeAnyPersistenceWrite(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo, &fakeRouter{})
	_, err := svc.Create(context.Background(), uuid.New(), CreateConnectionInput{
		ProviderKey: "custom",
		Name:        "Missing endpoint",
		APIKey:      "must-not-be-stored",
	})
	if !provider.IsEndpointValidationError(err) {
		t.Fatalf("Create error = %v, want endpoint validation error", err)
	}
	if repo.orgCalls != 0 || repo.createCalls != 0 || repo.upsertedKey != "" {
		t.Fatalf("persistence touched after missing endpoint: %#v", repo)
	}
}

func TestTestSuccess(t *testing.T) {
	workspaceID := uuid.New()
	accountID := uuid.New()
	router := &fakeRouter{invokeResp: provider.Response{ProviderModelID: "gpt-4.1-mini"}}
	repo := &fakeRepo{secrets: map[string]string{"PROVIDER_OPENAI_API_KEY": "sk-test-secret"}}
	svc := NewService(repo, router)

	result, err := svc.Test(context.Background(), repository.ProviderAccountRow{
		ID:                  accountID,
		WorkspaceID:         &workspaceID,
		ProviderKey:         "openai",
		CredentialReference: "workspace-secret://PROVIDER_OPENAI_API_KEY",
		BaseURL:             "https://models.example.com/v1",
		Status:              "active",
	}, TestInput{})
	if err != nil {
		t.Fatalf("Test: %v", err)
	}
	if !result.Passed || result.Status != "passed" {
		t.Fatalf("result = %#v, want passed", result)
	}
	if router.lastInvoke.ProviderAccountID != accountID.String() || router.lastInvoke.Model != "gpt-4.1-mini" || router.lastInvoke.BaseURL != "https://models.example.com/v1" {
		t.Fatalf("invoke request = %#v", router.lastInvoke)
	}
}

func TestTestFailureIsSanitized(t *testing.T) {
	workspaceID := uuid.New()
	secret := "sk-live-leaky-value"
	router := &fakeRouter{invokeErr: provider.NewFailure("openai", provider.FailureCodeAuth, "bad key "+secret+" from workspace-secret://PROVIDER_OPENAI_API_KEY", false, errors.New("auth"))}
	repo := &fakeRepo{secrets: map[string]string{"PROVIDER_OPENAI_API_KEY": secret}}
	svc := NewService(repo, router)

	result, err := svc.Test(context.Background(), repository.ProviderAccountRow{
		ID:                  uuid.New(),
		WorkspaceID:         &workspaceID,
		ProviderKey:         "openai",
		CredentialReference: "workspace-secret://PROVIDER_OPENAI_API_KEY",
		Status:              "active",
	}, TestInput{Model: "custom-model"})
	if err != nil {
		t.Fatalf("Test: %v", err)
	}
	if result.Passed || result.Code != string(provider.FailureCodeAuth) {
		t.Fatalf("result = %#v, want failed auth", result)
	}
	if strings.Contains(result.Message, secret) || strings.Contains(result.Message, "workspace-secret://") {
		t.Fatalf("message not sanitized: %q", result.Message)
	}
	if !strings.Contains(result.Message, "[redacted]") || !strings.Contains(result.Message, "[credential-reference]") {
		t.Fatalf("message missing redaction markers: %q", result.Message)
	}
}

func TestTestSanitizesEnvCredential(t *testing.T) {
	workspaceID := uuid.New()
	secret := "env-secret-value"
	t.Setenv("OPENAI_API_KEY", secret)
	router := &fakeRouter{invokeErr: provider.NewFailure("openai", provider.FailureCodeAuth, "bad key "+secret, false, errors.New("auth"))}
	svc := NewService(&fakeRepo{}, router)

	result, err := svc.Test(context.Background(), repository.ProviderAccountRow{
		ID:                  uuid.New(),
		WorkspaceID:         &workspaceID,
		ProviderKey:         "openai",
		CredentialReference: "env://OPENAI_API_KEY",
		Status:              "active",
	}, TestInput{})
	if err != nil {
		t.Fatalf("Test: %v", err)
	}
	if strings.Contains(result.Message, secret) || !strings.Contains(result.Message, "[redacted]") {
		t.Fatalf("message not sanitized: %q", result.Message)
	}
}

func TestTestUnconfiguredRouter(t *testing.T) {
	svc := NewService(&fakeRepo{}, nil)
	result, err := svc.Test(context.Background(), repository.ProviderAccountRow{ProviderKey: "openai"}, TestInput{})
	if err != nil {
		t.Fatalf("Test: %v", err)
	}
	if result.Passed || result.Code != string(provider.FailureCodeUnsupportedProvider) {
		t.Fatalf("result = %#v, want unconfigured failure", result)
	}
}

func TestCustomSmokeTestRequiresExplicitModel(t *testing.T) {
	workspaceID := uuid.New()
	router := &fakeRouter{}
	svc := NewService(&fakeRepo{}, router)
	result, err := svc.Test(context.Background(), repository.ProviderAccountRow{
		ID:          uuid.New(),
		WorkspaceID: &workspaceID,
		ProviderKey: "custom",
		BaseURL:     "https://models.example.com/v1",
		Status:      "active",
	}, TestInput{})
	if err != nil {
		t.Fatalf("Test: %v", err)
	}
	if result.Passed || result.Model != "" || router.invokeCalls != 0 || !strings.Contains(result.Message, "pass --model") {
		t.Fatalf("result/router calls = %#v/%d", result, router.invokeCalls)
	}
}

func TestListModelsCachesAndServesStaleOnError(t *testing.T) {
	workspaceID := uuid.New()
	account := repository.ProviderAccountRow{
		ID:                  uuid.New(),
		WorkspaceID:         &workspaceID,
		ProviderKey:         "openai",
		CredentialReference: "workspace-secret://PROVIDER_OPENAI_API_KEY",
		BaseURL:             "https://models.example.com/v1",
		Status:              "active",
	}
	router := &fakeRouter{models: []provider.ModelInfo{{ID: "gpt-4.1"}}}
	svc := NewService(&fakeRepo{secrets: map[string]string{"PROVIDER_OPENAI_API_KEY": "k"}}, router)

	first, err := svc.ListModels(context.Background(), account)
	if err != nil || len(first) != 1 {
		t.Fatalf("first ListModels: %v %v", first, err)
	}
	// Second call should be served from cache (no extra router call).
	if _, err := svc.ListModels(context.Background(), account); err != nil {
		t.Fatalf("cached ListModels: %v", err)
	}
	if router.listCalls != 1 {
		t.Fatalf("router list calls = %d, want 1 (cache hit)", router.listCalls)
	}
	if router.lastList.BaseURL != account.BaseURL {
		t.Fatalf("ListModels base URL = %q, want %q", router.lastList.BaseURL, account.BaseURL)
	}

	// Force a refetch and make it fail; stale cache should still be returned.
	svc.cache.invalidate(account.ID)
	svc.cache.set(account.ID, []provider.ModelInfo{{ID: "stale"}})
	svc.cache.entries[account.ID] = modelsCacheEntry{models: []provider.ModelInfo{{ID: "stale"}}} // expired (fetchedAt zero)
	router.modelsErr = errors.New("provider down")
	router.models = nil
	got, err := svc.ListModels(context.Background(), account)
	if err != nil {
		t.Fatalf("stale ListModels returned error: %v", err)
	}
	if len(got) != 1 || got[0].ID != "stale" {
		t.Fatalf("want stale entry, got %#v", got)
	}
}

func TestListModelsUnconfiguredRouterReturnsFailure(t *testing.T) {
	svc := NewService(&fakeRepo{}, nil)
	_, err := svc.ListModels(context.Background(), repository.ProviderAccountRow{ProviderKey: "openai"})
	if err == nil {
		t.Fatal("ListModels returned nil error with an unconfigured router")
	}
	failure, ok := provider.AsFailure(err)
	if !ok || failure.Code != provider.FailureCodeUnsupportedCapability {
		t.Fatalf("ListModels error = %v, want unsupported-capability failure", err)
	}
}
