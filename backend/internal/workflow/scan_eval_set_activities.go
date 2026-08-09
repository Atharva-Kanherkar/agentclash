package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/agentclash/agentclash/backend/internal/repository"
	"github.com/agentclash/agentclash/runtime/provider"
	"github.com/agentclash/agentclash/runtime/scanners"
	"github.com/google/uuid"
	"go.temporal.io/sdk/activity"
)

const (
	listScanTargetsActivityName = "workflow.list_scan_targets"
	scanOneTargetActivityName  = "workflow.scan_one_target"
	// Deprecated name kept registered for any in-flight histories that scheduled the
	// pre-fanout single-shot activity (new workflow is greenfield).
	scanEvalSetActivityName = "workflow.scan_eval_set"
)

type ScanFindingRepository interface {
	UpsertScanFinding(ctx context.Context, params repository.UpsertScanFindingParams) (repository.ScanFinding, error)
	ClearScanFindingsForTarget(ctx context.Context, evalSetID uuid.UUID, caseKey, scanner, scannerVersion string) error
	ListCaseResults(ctx context.Context, filter repository.ListCaseResultsFilter) ([]repository.CaseResult, error)
	GetCaseResultByID(ctx context.Context, id uuid.UUID) (repository.CaseResult, error)
}

type ListScanTargetsInput struct {
	EvalSetID uuid.UUID `json:"eval_set_id"`
	Scanners  []string  `json:"scanners"`
}

// ScanTarget is a compact (case × scanner) unit — no transcript in workflow history.
type ScanTarget struct {
	EvalSetID      uuid.UUID `json:"eval_set_id"`
	WorkspaceID    uuid.UUID `json:"workspace_id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	CaseResultID   uuid.UUID `json:"case_result_id"`
	CaseKey        string    `json:"case_key"`
	MatrixKey      string    `json:"matrix_key"`
	Scanner        string    `json:"scanner"`
}

func (a *Activities) WithScanFindingRepository(repo ScanFindingRepository) *Activities {
	a.scanFindingRepo = repo
	return a
}

func (a *Activities) ListScanTargets(ctx context.Context, input ListScanTargetsInput) ([]ScanTarget, error) {
	if a.evalSetRepo == nil || a.scanFindingRepo == nil {
		return nil, errors.New("scan repositories are not configured")
	}
	if info := activity.GetInfo(ctx); info.TaskQueue == TaskQueueExecution || info.TaskQueue == TaskQueueScoring {
		return nil, fmt.Errorf("scan activity must not run on %q queue", info.TaskQueue)
	}
	set, err := a.evalSetRepo.GetEvalSetByID(ctx, input.EvalSetID)
	if err != nil {
		return nil, wrapActivityError(err)
	}
	names := input.Scanners
	if len(names) == 0 {
		names = scannersFromManifest(set.Manifest)
	}
	if len(names) == 0 {
		return nil, nil
	}
	for _, name := range names {
		if _, err := scanners.LookupBuiltIn(strings.TrimSpace(name)); err != nil {
			return nil, wrapActivityError(err)
		}
	}

	targets := make([]ScanTarget, 0)
	var cursor *uuid.UUID
	for {
		cases, err := a.scanFindingRepo.ListCaseResults(ctx, repository.ListCaseResultsFilter{
			WorkspaceID: set.WorkspaceID,
			EvalSetID:   input.EvalSetID,
			CursorID:    cursor,
			Limit:       200,
		})
		if err != nil {
			return nil, wrapActivityError(err)
		}
		if len(cases) == 0 {
			break
		}
		for _, c := range cases {
			for _, name := range names {
				targets = append(targets, ScanTarget{
					EvalSetID:      set.ID,
					WorkspaceID:    set.WorkspaceID,
					OrganizationID: set.OrganizationID,
					CaseResultID:   c.ID,
					CaseKey:        c.CaseKey,
					MatrixKey:      c.MatrixKey,
					Scanner:        strings.TrimSpace(name),
				})
			}
			id := c.ID
			cursor = &id
		}
		if len(cases) < 200 {
			break
		}
	}
	return targets, nil
}

func (a *Activities) ScanOneTarget(ctx context.Context, target ScanTarget) error {
	if a.scanFindingRepo == nil {
		return errors.New("scan finding repository is not configured")
	}
	if info := activity.GetInfo(ctx); info.TaskQueue == TaskQueueExecution || info.TaskQueue == TaskQueueScoring {
		return fmt.Errorf("scan activity must not run on %q queue", info.TaskQueue)
	}
	def, err := scanners.LookupBuiltIn(target.Scanner)
	if err != nil {
		return wrapActivityError(err)
	}
	c, err := a.scanFindingRepo.GetCaseResultByID(ctx, target.CaseResultID)
	if err != nil {
		return wrapActivityError(err)
	}
	return a.runOneScanner(ctx, target, c, def)
}

// ScanEvalSet remains as a compatibility progressive-batch path (tests / legacy).
func (a *Activities) ScanEvalSet(ctx context.Context, input ScanEvalSetActivityInput) error {
	targets, err := a.ListScanTargets(ctx, ListScanTargetsInput{EvalSetID: input.EvalSetID, Scanners: input.Scanners})
	if err != nil {
		return err
	}
	for _, t := range targets {
		if err := a.ScanOneTarget(ctx, t); err != nil {
			return err
		}
	}
	return nil
}

type ScanEvalSetActivityInput struct {
	EvalSetID uuid.UUID `json:"eval_set_id"`
	Scanners  []string  `json:"scanners"`
}

func (a *Activities) runOneScanner(ctx context.Context, target ScanTarget, c repository.CaseResult, def scanners.Definition) error {
	caseID := c.ID
	switch def.Kind {
	case scanners.KindPattern:
		findings, err := scanners.RunPattern(def, c.TranscriptText)
		if err != nil {
			return wrapActivityError(err)
		}
		if len(findings) == 0 {
			// Rescan with no hits must drop stale rows from prior scans.
			if err := a.scanFindingRepo.ClearScanFindingsForTarget(ctx, target.EvalSetID, c.CaseKey, def.Name, def.Version); err != nil {
				return wrapActivityError(err)
			}
			return nil
		}
		for _, f := range findings {
			if _, err := a.scanFindingRepo.UpsertScanFinding(ctx, repository.UpsertScanFindingParams{
				WorkspaceID:    target.WorkspaceID,
				OrganizationID: target.OrganizationID,
				EvalSetID:      target.EvalSetID,
				CaseResultID:   &caseID,
				MatrixKey:      c.MatrixKey,
				CaseKey:        c.CaseKey,
				Scanner:        f.Scanner,
				ScannerVersion: f.Version,
				Severity:       string(f.Severity),
				Category:       f.Category,
				Evidence:       f.Evidence,
				Confidence:     f.Confidence,
				Status:         "open",
			}); err != nil {
				return wrapActivityError(err)
			}
		}
	case scanners.KindLLM:
		raw, err := a.invokeScannerLLM(ctx, def, c.TranscriptText)
		if err != nil {
			return wrapActivityError(err)
		}
		verdict, err := scanners.ParseLLMVerdict(raw, def.LLM.SchemaVersion)
		if err != nil {
			// Malformed → retryable application error (never store raw).
			return wrapActivityError(fmt.Errorf("malformed llm scanner verdict: %w", err))
		}
		if f := scanners.FindingFromLLM(def, verdict); f != nil {
			if _, err := a.scanFindingRepo.UpsertScanFinding(ctx, repository.UpsertScanFindingParams{
				WorkspaceID:    target.WorkspaceID,
				OrganizationID: target.OrganizationID,
				EvalSetID:      target.EvalSetID,
				CaseResultID:   &caseID,
				MatrixKey:      c.MatrixKey,
				CaseKey:        c.CaseKey,
				Scanner:        f.Scanner,
				ScannerVersion: f.Version,
				Severity:       string(f.Severity),
				Category:       f.Category,
				Evidence:       f.Evidence,
				Confidence:     f.Confidence,
				Status:         "open",
			}); err != nil {
				return wrapActivityError(err)
			}
		} else {
			// hit:false — clear any prior finding for this scanner/target.
			if err := a.scanFindingRepo.ClearScanFindingsForTarget(ctx, target.EvalSetID, c.CaseKey, def.Name, def.Version); err != nil {
				return wrapActivityError(err)
			}
		}
	}
	return nil
}

func (a *Activities) invokeScannerLLM(ctx context.Context, def scanners.Definition, transcript string) ([]byte, error) {
	if a.scannerLLM != nil {
		return a.scannerLLM(ctx, def, transcript)
	}
	if a.judgeClient == nil {
		return []byte(`{"schema_version":1,"hit":false,"severity":"low","category":"","evidence":"","confidence":0}`), nil
	}
	prompt := def.LLM.Prompt + "\n\nTRANSCRIPT:\n" + transcript
	resp, err := a.judgeClient.InvokeModel(ctx, provider.Request{
		Messages: []provider.Message{{Role: "user", Content: prompt}},
	})
	if err != nil {
		return nil, err
	}
	return []byte(resp.OutputText), nil
}

type scannerLLMFunc func(ctx context.Context, def scanners.Definition, transcript string) ([]byte, error)

func (a *Activities) WithScannerLLM(fn scannerLLMFunc) *Activities {
	a.scannerLLM = fn
	return a
}

func scannersFromManifest(manifest json.RawMessage) []string {
	var parsed struct {
		Scanners []string `json:"scanners"`
	}
	if err := json.Unmarshal(manifest, &parsed); err != nil {
		return nil
	}
	return parsed.Scanners
}
