import type { ArenaLaneState } from "@/hooks/use-agent-arena";
import type { RunAgent } from "@/lib/api/types";

/**
 * Target step count used as the denominator for the track and each lane's
 * progress bar. Track + lane must agree on this value or the two panels
 * render the same agent at different percentages.
 *
 * We don't currently have the challenge's real step budget on the client
 * (`RuntimeProfile.MaxIterations` lives backend-side and isn't on the
 * `RunAgent` payload), so the scale is derived purely from observed data:
 * the frontier = the furthest-advanced agent, plus a 1-step buffer so the
 * leader's dot never pins at the finish line while still racing.
 *
 * When nothing has happened yet (all agents at step 0) we return 1 to keep
 * division safe.
 */
export function computeTargetSteps(
  agents: RunAgent[],
  lanes: Record<string, ArenaLaneState>,
): number {
  let max = 0;
  for (const a of agents) {
    const lane = lanes[a.id];
    if (lane && lane.stepIndex > max) max = lane.stepIndex;
  }
  return Math.max(max + 1, 1);
}

/**
 * Ranks agents by: (1) non-failed first, (2) higher stepIndex first,
 * (3) more model calls as tiebreak, (4) lane_index ascending.
 *
 * Used by both `RaceTrack` and `RaceModeArena` — must live in exactly one
 * place so the two views never disagree on who's ahead.
 */
export function rankAgents(
  agents: RunAgent[],
  lanes: Record<string, ArenaLaneState>,
): { agent: RunAgent; position: number }[] {
  const sorted = [...agents].sort((a, b) => {
    const aFailed = a.status === "failed" ? 1 : 0;
    const bFailed = b.status === "failed" ? 1 : 0;
    if (aFailed !== bFailed) return aFailed - bFailed;
    const aStep = lanes[a.id]?.stepIndex ?? 0;
    const bStep = lanes[b.id]?.stepIndex ?? 0;
    if (aStep !== bStep) return bStep - aStep;
    const aCalls = lanes[a.id]?.modelCalls ?? 0;
    const bCalls = lanes[b.id]?.modelCalls ?? 0;
    if (aCalls !== bCalls) return bCalls - aCalls;
    return a.lane_index - b.lane_index;
  });
  return sorted.map((agent, i) => ({ agent, position: i + 1 }));
}

/**
 * A single source of truth for "is this lane the leader?". Before the run
 * finishes (no winnerAgentId), we only flag position-1 as leader when the
 * agent is actually racing — otherwise a queued-but-first-in-order agent
 * would wear the green leader stripe before it's even started.
 */
export function deriveLeader(
  agent: RunAgent,
  position: number,
  winnerAgentId: string | undefined,
): boolean {
  if (winnerAgentId) return agent.id === winnerAgentId;
  if (position !== 1) return false;
  return agent.status === "executing" || agent.status === "evaluating";
}
