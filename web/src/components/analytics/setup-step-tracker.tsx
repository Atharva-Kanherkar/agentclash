"use client";

import { useEffect } from "react";
import { captureWebEvent } from "@/lib/analytics/posthog-client";
import { WEB_EVENTS } from "@/lib/analytics/events";

export type SetupSurface =
  | "onboarding_wizard"
  | "activation_banner"
  | "activation_checklist";

interface SetupStepProperties {
  step: string;
  surface: SetupSurface;
  workspaceId?: string;
}

export function SetupStepView({
  step,
  surface,
  workspaceId,
}: SetupStepProperties) {
  useEffect(() => {
    captureWebEvent(WEB_EVENTS.SETUP_STEP_VIEWED, {
      step,
      surface,
      ...(workspaceId ? { workspace_id: workspaceId } : {}),
    });
  }, [step, surface, workspaceId]);
  return null;
}

export function trackSetupStepClick({
  step,
  surface,
  workspaceId,
  action,
}: SetupStepProperties & { action: string }): void {
  captureWebEvent(WEB_EVENTS.SETUP_STEP_CLICKED, {
    step,
    surface,
    action,
    ...(workspaceId ? { workspace_id: workspaceId } : {}),
  });
}
