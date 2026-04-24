"use client";

import Link from "next/link";
import { Check, Play, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { useOnboardingState } from "./use-onboarding-state";

interface WorkspaceWelcomeProps {
  workspaceId: string;
  deploymentsCount: number;
  packsCount: number;
  runsCount: number;
  onOpenCreateRun: () => void;
}

export function WorkspaceWelcome({
  workspaceId,
  deploymentsCount,
  packsCount,
  runsCount,
  onOpenCreateRun,
}: WorkspaceWelcomeProps) {
  const { dismissed, dismiss } = useOnboardingState(workspaceId);

  const step1Done = deploymentsCount >= 2;
  const step2Done = packsCount >= 1;
  const step3Done = runsCount >= 1;
  const allDone = step1Done && step2Done && step3Done;

  if (dismissed) return null;
  // Once the user has a run, the page shows its usual content — the welcome
  // retires itself even if they never clicked dismiss.
  if (allDone) return null;

  const steps: Array<{
    title: string;
    description: string;
    done: boolean;
    href?: string;
    onClick?: () => void;
    cta: string;
  }> = [
    {
      title: "Deploy two agents",
      description:
        "The competitors you'll compare. A deployment is a model wired to a runtime.",
      done: step1Done,
      href: `/workspaces/${workspaceId}/deployments`,
      cta: step1Done ? "Manage deployments" : "Go to deployments",
    },
    {
      title: "Pick a challenge pack",
      description:
        "The task each agent will attempt. Publish your own or browse the catalog.",
      done: step2Done,
      href: `/workspaces/${workspaceId}/challenge-packs`,
      cta: step2Done ? "Browse packs" : "Browse packs",
    },
    {
      title: "Run your first clash",
      description:
        "Start the head-to-head and open the replay to see each step.",
      done: step3Done,
      onClick: onOpenCreateRun,
      cta: "Run your first clash",
    },
  ];

  const firstIncomplete = steps.findIndex((s) => !s.done);

  return (
    <section
      role="region"
      aria-label="Getting started with AgentClash"
      className="relative mb-6 rounded-xl border border-border bg-card/50 p-6 animate-in fade-in-0 duration-200"
    >
      <button
        type="button"
        onClick={dismiss}
        aria-label="Dismiss onboarding"
        className="absolute right-3 top-3 inline-flex size-7 items-center justify-center rounded-md text-muted-foreground/70 transition-colors hover:bg-muted hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/50"
      >
        <X className="size-4" />
      </button>

      <div className="mb-5 max-w-2xl pr-8">
        <h2 className="text-base font-semibold tracking-tight text-foreground">
          Get started with AgentClash
        </h2>
        <p className="mt-1 text-sm text-muted-foreground">
          AgentClash compares agents by making them solve the same task with
          the same tools and constraints. Live scores and replays show you
          which agent actually performs better.
        </p>
      </div>

      <ol className="mb-5 space-y-1.5">
        {steps.map((step, i) => {
          const emphasized = i === firstIncomplete;
          return (
            <li
              key={step.title}
              className={cn(
                "flex items-start gap-3 rounded-lg border px-3 py-2.5 transition-colors",
                step.done
                  ? "border-transparent bg-muted/10"
                  : emphasized
                    ? "border-border/80 bg-muted/30"
                    : "border-transparent",
              )}
            >
              <span
                aria-hidden="true"
                className={cn(
                  "mt-0.5 flex size-5 shrink-0 items-center justify-center rounded-full",
                  step.done
                    ? "bg-foreground text-background"
                    : emphasized
                      ? "border border-foreground/40 bg-background text-foreground"
                      : "border border-border text-muted-foreground",
                )}
              >
                {step.done ? (
                  <Check className="size-3" />
                ) : (
                  <span className="text-[10px] font-semibold leading-none">
                    {i + 1}
                  </span>
                )}
              </span>
              <div className="min-w-0 flex-1">
                <div className="flex items-center gap-2">
                  <span
                    className={cn(
                      "text-sm font-medium",
                      step.done
                        ? "text-muted-foreground"
                        : "text-foreground",
                    )}
                  >
                    {step.done && (
                      <span className="sr-only">Completed: </span>
                    )}
                    {step.title}
                  </span>
                </div>
                {!step.done && (
                  <p className="mt-0.5 text-xs text-muted-foreground">
                    {step.description}
                  </p>
                )}
                {emphasized && !step.done && (
                  <div className="mt-2">
                    {step.onClick ? (
                      <Button
                        size="xs"
                        variant="outline"
                        onClick={step.onClick}
                      >
                        {step.cta}
                      </Button>
                    ) : step.href ? (
                      <Button
                        size="xs"
                        variant="outline"
                        render={<Link href={step.href} />}
                      >
                        {step.cta}
                      </Button>
                    ) : null}
                  </div>
                )}
              </div>
            </li>
          );
        })}
      </ol>

      <div className="flex flex-wrap items-center gap-2">
        <Button size="sm" onClick={onOpenCreateRun}>
          <Play data-icon="inline-start" className="size-4" />
          Run your first clash
        </Button>
        <Button
          size="sm"
          variant="ghost"
          render={
            <Link href="/manifesto" target="_blank" rel="noreferrer" />
          }
        >
          How it works →
        </Button>
      </div>
    </section>
  );
}
