"use client";

import { useMemo } from "react";

import type { RunAgent } from "@/lib/api/types";
import type { ArenaLaneState } from "@/hooks/use-agent-arena";
import { EMPTY_LANE } from "@/hooks/use-agent-arena";
import type { CommentaryEntry } from "@/hooks/use-agent-commentary";

import { RaceCommentary } from "./race-commentary";
import { RaceLane } from "./race-lane";
import { RaceTrack } from "./race-track";

import "./race-mode.css";

interface RaceModeArenaProps {
  agents: RunAgent[];
  lanes: Record<string, ArenaLaneState>;
  workspaceId: string;
  runId: string;
  winnerAgentId?: string;
  showCommentary: boolean;
  commentaryEntries: CommentaryEntry[];
  isActive: boolean;
  /** Map of agent.id → terminal-only footer node (scorecard, etc.). */
  laneFooters?: Record<string, React.ReactNode>;
}

export function RaceModeArena({
  agents,
  lanes,
  workspaceId,
  runId,
  winnerAgentId,
  showCommentary,
  commentaryEntries,
  isActive,
  laneFooters,
}: RaceModeArenaProps) {
  const ranked = useMemo(() => rankAgents(agents, lanes), [agents, lanes]);

  return (
    <div className="race-mode-root">
      <RaceTrack
        agents={agents}
        lanes={lanes}
        winnerAgentId={winnerAgentId}
      />

      <div
        className={`rm-grid${showCommentary ? " rm-grid--with-booth" : ""}`}
      >
        <div className="rm-lanes">
          {ranked.map(({ agent, position }) => (
            <RaceLane
              key={agent.id}
              agent={agent}
              lane={lanes[agent.id] ?? EMPTY_LANE}
              position={position}
              isWinner={
                winnerAgentId
                  ? agent.id === winnerAgentId
                  : position === 1 && agent.status !== "failed"
              }
              workspaceId={workspaceId}
              runId={runId}
              footer={laneFooters?.[agent.id] ?? null}
            />
          ))}
        </div>
        {showCommentary && (
          <RaceCommentary
            entries={commentaryEntries}
            isActive={isActive}
          />
        )}
      </div>
    </div>
  );
}

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
