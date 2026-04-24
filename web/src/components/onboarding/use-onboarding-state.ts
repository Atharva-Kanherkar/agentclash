"use client";

import { useCallback, useSyncExternalStore } from "react";

const KEY_PREFIX = "agentclash:onboarding";
const SYNC_EVENT = "agentclash:onboarding:sync";

export const onboardingStorageKeys = {
  dismissed: (workspaceId: string) => `${KEY_PREFIX}:dismissed:${workspaceId}`,
  firstRunSeen: (workspaceId: string) =>
    `${KEY_PREFIX}:first_run_seen:${workspaceId}`,
};

function readFlag(key: string): boolean {
  if (typeof window === "undefined") return false;
  try {
    return window.localStorage.getItem(key) === "1";
  } catch {
    return false;
  }
}

function writeFlag(key: string, value: boolean) {
  if (typeof window === "undefined") return;
  try {
    if (value) window.localStorage.setItem(key, "1");
    else window.localStorage.removeItem(key);
  } catch {
    return;
  }
  window.dispatchEvent(new CustomEvent(SYNC_EVENT));
}

function subscribe(onChange: () => void): () => void {
  window.addEventListener("storage", onChange);
  window.addEventListener(SYNC_EVENT, onChange);
  return () => {
    window.removeEventListener("storage", onChange);
    window.removeEventListener(SYNC_EVENT, onChange);
  };
}

const serverSnapshot = () => false;

export interface OnboardingState {
  dismissed: boolean;
  firstRunSeen: boolean;
  dismiss: () => void;
  restore: () => void;
  markFirstRunSeen: () => void;
}

export function useOnboardingState(workspaceId: string): OnboardingState {
  const dismissed = useSyncExternalStore(
    subscribe,
    () => readFlag(onboardingStorageKeys.dismissed(workspaceId)),
    serverSnapshot,
  );
  const firstRunSeen = useSyncExternalStore(
    subscribe,
    () => readFlag(onboardingStorageKeys.firstRunSeen(workspaceId)),
    serverSnapshot,
  );

  const dismiss = useCallback(() => {
    writeFlag(onboardingStorageKeys.dismissed(workspaceId), true);
  }, [workspaceId]);

  const restore = useCallback(() => {
    writeFlag(onboardingStorageKeys.dismissed(workspaceId), false);
    writeFlag(onboardingStorageKeys.firstRunSeen(workspaceId), false);
  }, [workspaceId]);

  const markFirstRunSeen = useCallback(() => {
    writeFlag(onboardingStorageKeys.firstRunSeen(workspaceId), true);
  }, [workspaceId]);

  return { dismissed, firstRunSeen, dismiss, restore, markFirstRunSeen };
}

export function restartOnboarding(workspaceId: string) {
  writeFlag(onboardingStorageKeys.dismissed(workspaceId), false);
  writeFlag(onboardingStorageKeys.firstRunSeen(workspaceId), false);
}
