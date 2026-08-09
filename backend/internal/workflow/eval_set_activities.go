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
	ListRunAgentsByRunID(ctx context.Context, runID uuid.UUID) ([]domain.RunAgent, error)
	GetRunAgentScorecardByRunAgentID(ctx context.Context, runAgentID uuid.UUID) (repository.RunAgentScorecard, error)
	UpsertCaseResult(ctx context.Context, params repository.UpsertCaseResultParams) (repository.CaseResult, error)
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
		WorkspaceID:       set.WorkspaceID,
		MaxConcurrentRuns: set.MaxConcurrentRuns,
		Status:            string(set.Status),
		BudgetUSD:         set.BudgetUSD,
		SpentUSD:          set.SpentUSD,
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
	set, err := a.evalSetRepo.GetEvalSetByID(ctx, evalSetID)
	if err != nil {
		return repository.EvalSetResult{}, wrapActivityError(err)
	}
	rows := make([]comboRow, 0)
	perPack := map[string]int{}
	runCount := int32(0)
	for i, id := range ids {
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
		sessionID := id
		for _, run := range runs {
			matrixKey := seriesMatrixKeyFromPlan(run.ExecutionPlan)
			if matrixKey != "" {
				rows = append(rows, comboRow{
					MatrixKey: matrixKey,
					PackRef:   pack,
					Status:    string(run.Status),
				})
			}
			agents, agentErr := a.evalSetRepo.ListRunAgentsByRunID(ctx, run.ID)
			if agentErr != nil {
				return repository.EvalSetResult{}, wrapActivityError(agentErr)
			}
			for _, agent := range agents {
				caseKey := matrixKey
				if caseKey == "" {
					caseKey = agent.ID.String()
				}
				verdict, correctness, score := caseResultOutcomeFromAgent(ctx, a.evalSetRepo, agent)
				transcript := "run " + run.ID.String() + " agent " + agent.ID.String() + " status " + string(agent.Status)
				if matrixKey != "" {
					transcript += " matrix_key " + matrixKey
				}
				deploymentID := agent.AgentDeploymentID
				_, _ = a.evalSetRepo.UpsertCaseResult(ctx, repository.UpsertCaseResultParams{
					WorkspaceID:       set.WorkspaceID,
					OrganizationID:    set.OrganizationID,
					EvalSetID:         &evalSetID,
					EvalSessionID:     &sessionID,
					RunID:             run.ID,
					RunAgentID:        agent.ID,
					MatrixKey:         matrixKey,
					PackRef:           pack,
					CaseKey:           caseKey,
					AgentDeploymentID: &deploymentID,
					Score:             score,
					Verdict:           verdict,
					Correctness:       correctness,
					TranscriptText:    transcript,
				})
			}
		}
	}
	agg := map[string]any{
		"sessions":     len(ids),
		"runs":         runCount,
		"combinations": rows,
		"per_pack":     perPack,
		"spent_usd":    set.SpentUSD,
	}
	if set.BudgetUSD != nil {
		agg["budget_usd"] = *set.BudgetUSD
		if set.SpentUSD >= *set.BudgetUSD {
			agg["outcome"] = "budget_exceeded"
			agg["partial_results"] = true
		}
	}
	aggregate, _ := json.Marshal(agg)
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

// caseResultOutcomeFromAgent derives warehouse verdict/correctness from the
// agent scorecard when present, otherwise from the agent status. Parent run
// completion alone must not mark an agent as pass.
func caseResultOutcomeFromAgent(ctx context.Context, repo EvalSetRepository, agent domain.RunAgent) (verdict string, correctness *bool, score *float64) {
	if repo != nil {
		scorecard, err := repo.GetRunAgentScorecardByRunAgentID(ctx, agent.ID)
		if err == nil {
			score = scorecard.OverallScore
			if scorecard.Passed != nil {
				if *scorecard.Passed {
					verdict = "pass"
				} else {
					verdict = "fail"
				}
				correctness = scorecard.Passed
				return verdict, correctness, score
			}
		} else if !errors.Is(err, repository.ErrRunAgentScorecardNotFound) {
			// Non-not-found scorecard errors: fall through to agent status rather
			// than failing aggregation; projection stays best-effort.
		}
	}
	switch agent.Status {
	case domain.RunAgentStatusCompleted:
		verdict = "pass"
		t := true
		correctness = &t
	case domain.RunAgentStatusFailed:
		verdict = "fail"
		f := false
		correctness = &f
	}
	return verdict, correctness, score
}
