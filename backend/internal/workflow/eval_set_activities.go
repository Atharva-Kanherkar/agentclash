package workflow

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/agentclash/agentclash/backend/internal/repository"
	"github.com/agentclash/agentclash/runtime/domain"
	"github.com/google/uuid"
)

const (
	transitionEvalSetStatusActivityName = "workflow.transition_eval_set_status"
	loadEvalSetActivityName             = "workflow.load_eval_set"
	listEvalSetSessionIDsActivityName   = "workflow.list_eval_set_session_ids"
	aggregateEvalSetActivityName        = "workflow.aggregate_eval_set"
)

type EvalSetRepository interface {
	GetEvalSetByID(ctx context.Context, id uuid.UUID) (repository.EvalSet, error)
	TransitionEvalSetStatus(ctx context.Context, id uuid.UUID, from, to domain.EvalSetStatus, failureReason *string) (repository.EvalSet, error)
	ListEvalSessionsByEvalSetID(ctx context.Context, evalSetID uuid.UUID) ([]uuid.UUID, []string, error)
	UpsertEvalSetResult(ctx context.Context, evalSetID uuid.UUID, aggregate, evidence json.RawMessage, sessionCount, runCount int32) (repository.EvalSetResult, error)
	GetEvalSessionByID(ctx context.Context, id uuid.UUID) (domain.EvalSession, error)
	ListRunsByEvalSessionID(ctx context.Context, evalSessionID uuid.UUID) ([]domain.Run, error)
}

type TransitionEvalSetStatusInput struct {
	EvalSetID     uuid.UUID            `json:"eval_set_id"`
	From          domain.EvalSetStatus `json:"from"`
	To            domain.EvalSetStatus `json:"to"`
	FailureReason *string              `json:"failure_reason,omitempty"`
}

func (a *Activities) WithEvalSetRepository(repo EvalSetRepository) *Activities {
	a.evalSetRepo = repo
	return a
}

func (a *Activities) TransitionEvalSetStatus(ctx context.Context, input TransitionEvalSetStatusInput) (repository.EvalSet, error) {
	if a.evalSetRepo == nil {
		return repository.EvalSet{}, errors.New("eval set repository is not configured")
	}
	set, err := a.evalSetRepo.TransitionEvalSetStatus(ctx, input.EvalSetID, input.From, input.To, input.FailureReason)
	return set, wrapActivityError(err)
}

func (a *Activities) LoadEvalSet(ctx context.Context, evalSetID uuid.UUID) (repositoryEvalSetView, error) {
	if a.evalSetRepo == nil {
		return repositoryEvalSetView{}, errors.New("eval set repository is not configured")
	}
	set, err := a.evalSetRepo.GetEvalSetByID(ctx, evalSetID)
	if err != nil {
		return repositoryEvalSetView{}, wrapActivityError(err)
	}
	return repositoryEvalSetView{
		ID:                set.ID,
		MaxConcurrentRuns: set.MaxConcurrentRuns,
		Status:            string(set.Status),
	}, nil
}

func (a *Activities) ListEvalSetSessionIDs(ctx context.Context, evalSetID uuid.UUID) ([]uuid.UUID, error) {
	if a.evalSetRepo == nil {
		return nil, errors.New("eval set repository is not configured")
	}
	ids, _, err := a.evalSetRepo.ListEvalSessionsByEvalSetID(ctx, evalSetID)
	if err != nil {
		return nil, wrapActivityError(err)
	}
	return ids, nil
}

func (a *Activities) AggregateEvalSet(ctx context.Context, evalSetID uuid.UUID) (repository.EvalSetResult, error) {
	if a.evalSetRepo == nil {
		return repository.EvalSetResult{}, errors.New("eval set repository is not configured")
	}
	ids, packs, err := a.evalSetRepo.ListEvalSessionsByEvalSetID(ctx, evalSetID)
	if err != nil {
		return repository.EvalSetResult{}, wrapActivityError(err)
	}

	type comboRow struct {
		MatrixKey string `json:"matrix_key,omitempty"`
		PackRef   string `json:"pack_ref"`
		Status    string `json:"status"`
	}
	rows := make([]comboRow, 0)
	perPack := map[string]int{}
	runCount := int32(0)
	for i, id := range ids {
		session, err := a.evalSetRepo.GetEvalSessionByID(ctx, id)
		if err != nil {
			return repository.EvalSetResult{}, wrapActivityError(err)
		}
		runs, err := a.evalSetRepo.ListRunsByEvalSessionID(ctx, id)
		if err != nil {
			return repository.EvalSetResult{}, wrapActivityError(err)
		}
		runCount += int32(len(runs))
		pack := ""
		if i < len(packs) {
			pack = packs[i]
		}
		perPack[pack]++
		rows = append(rows, comboRow{PackRef: pack, Status: string(session.Status)})
		for _, run := range runs {
			rows = append(rows, comboRow{
				MatrixKey: seriesMatrixKeyFromPlan(run.ExecutionPlan),
				PackRef:   pack,
				Status:    string(run.Status),
			})
		}
	}
	aggregate, _ := json.Marshal(map[string]any{
		"sessions":     len(ids),
		"runs":         runCount,
		"combinations": rows,
		"per_pack":     perPack,
	})
	evidence, _ := json.Marshal(map[string]any{"session_ids": ids})
	result, err := a.evalSetRepo.UpsertEvalSetResult(ctx, evalSetID, aggregate, evidence, int32(len(ids)), runCount)
	if err != nil {
		return repository.EvalSetResult{}, wrapActivityError(err)
	}
	return result, nil
}

func seriesMatrixKeyFromPlan(plan json.RawMessage) string {
	var parsed struct {
		Series *struct {
			MatrixKey string `json:"matrix_key"`
		} `json:"series"`
	}
	if err := json.Unmarshal(plan, &parsed); err != nil || parsed.Series == nil {
		return ""
	}
	return parsed.Series.MatrixKey
}
