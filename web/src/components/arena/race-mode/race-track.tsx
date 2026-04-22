"use client";

import { useMemo } from "react";
import type { RunAgent } from "@/lib/api/types";
import type { ArenaLaneState } from "@/hooks/use-agent-arena";

/**
 * Unified horizontal timeline — all agents plotted on the same 0→N-step axis,
 * racing against each other. Position is derived from each agent's live
 * stepIndex; failed agents stop at their last-seen step.
 *
 * We don't know the challenge's total step count from the client, so we scale
 * against max(observed stepIndex, DEFAULT_TARGET) — the track grows early and
 * compresses as the run progresses.
 */

const DEFAULT_TARGET_STEPS = 12;

interface RaceTrackProps {
  agents: RunAgent[];
  lanes: Record<string, ArenaLaneState>;
  winnerAgentId?: string;
}

export function RaceTrack({ agents, lanes, winnerAgentId }: RaceTrackProps) {
  const ordered = useMemo(() => rankAgents(agents, lanes), [agents, lanes]);
  const targetSteps = useMemo(() => {
    let max = DEFAULT_TARGET_STEPS;
    for (const a of agents) {
      const lane = lanes[a.id];
      if (lane && lane.stepIndex > max) max = lane.stepIndex;
    }
    return max;
  }, [agents, lanes]);

  const activeCount = agents.filter(
    (a) => a.status === "executing" || a.status === "evaluating",
  ).length;

  const maxStep = Math.max(
    ...agents.map((a) => lanes[a.id]?.stepIndex ?? 0),
    0,
  );

  return (
    <section className="rm-track" aria-label="Race track">
      <div className="rm-track__head">
        <h2 className="rm-track__title">Lap progress</h2>
        <div className="rm-track__meta">
          <span>
            {agents.length} {agents.length === 1 ? "lane" : "lanes"}
          </span>
          <span className="rm-sep">·</span>
          <span>
            {activeCount > 0
              ? `${activeCount} active`
              : maxStep > 0
                ? `step ${maxStep} / ${targetSteps}`
                : "standing by"}
          </span>
          <span className="rm-sep">·</span>
          <span className="rm-finish">finish</span>
        </div>
      </div>
      <div className="rm-track__body">
        {ordered.map(({ agent, position }) => {
          const lane = lanes[agent.id];
          const step = lane?.stepIndex ?? 0;
          const pct = clamp((step / Math.max(targetSteps, 1)) * 100, 0, 100);
          const isLeader = winnerAgentId
            ? agent.id === winnerAgentId
            : position === 1 &&
              (agent.status === "executing" || agent.status === "evaluating");
          const isFailed = agent.status === "failed";

          const rowClass = [
            "rm-track-row",
            isLeader && "rm-track-row--leader",
            isFailed && "rm-track-row--failed",
          ]
            .filter(Boolean)
            .join(" ");

          const wakeLeft = Math.max(pct - 18, 0);
          const wakeWidth = Math.min(pct, 18);

          return (
            <div key={agent.id} className={rowClass}>
              <span className="rm-track-row__pos">{position}</span>
              <span className="rm-track-row__name" title={agent.label}>
                {agent.label}
              </span>
              <div className="rm-track-row__lane">
                <div
                  className="rm-track-row__fill"
                  style={{ width: `${pct}%` }}
                />
                {!isFailed && wakeWidth > 0 && (
                  <span
                    className="rm-track-row__wake"
                    style={{
                      left: `${wakeLeft}%`,
                      width: `${wakeWidth}%`,
                    }}
                  />
                )}
                <span
                  className="rm-track-row__dot"
                  style={{ left: `${pct}%` }}
                />
              </div>
              <span className="rm-track-row__step">
                {isFailed ? "DNF" : `${step}/${targetSteps}`}
              </span>
            </div>
          );
        })}
      </div>
    </section>
  );
}

function clamp(n: number, lo: number, hi: number): number {
  return Math.min(Math.max(n, lo), hi);
}

/**
 * Ranks agents by: (1) non-failed first, (2) higher stepIndex first,
 * (3) more model calls as tiebreak, (4) lane_index ascending.
 */
function rankAgents(
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
