"use client";

import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { useAccessToken } from "@workos-inc/authkit-nextjs/components";
import { useCallback, useEffect, useRef, useState } from "react";
import {
  ArrowUp,
  Loader2,
  MessageSquarePlus,
  Paperclip,
  PanelRight,
  Square,
} from "lucide-react";
import { ClashMark } from "@/components/marketing/clash-mark";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogTitle,
} from "@/components/ui/dialog";
import { CreditsDialog } from "@/components/vibe/credits-dialog";
import { ArtifactPanel, ModelSelect } from "@/components/vibe/artifact-panel";
import { SafeMarkdown } from "@/components/vibe/safe-markdown";
import { VibeScorecard } from "@/components/vibe/scorecard";
import { createApiClient } from "@/lib/api/client";
import type { UserMeResponse } from "@/lib/api/types";
import {
  defaultModels,
  dollars,
  terminal,
  VibeError,
  vibeFetch,
  watchVibe,
  type CaseResult,
  type Models,
  type Operation,
  type Session,
  type VibeConfig,
} from "@/lib/vibe";

const starters = [
  "I have an agent that needs testing",
  "Help me build an agent",
  "I’m figuring out what AI could do for us",
];
export function VibeClient() {
  const params = useSearchParams();
  const { getAccessToken } = useAccessToken();
  const [session, setSession] = useState<Session | null>(null);
  const [config, setConfig] = useState<VibeConfig | null>(null);
  const [models, setModels] = useState<Models>(defaultModels);
  const [content, setContent] = useState("");
  const [pending, setPending] = useState(false);
  const [error, setError] = useState("");
  const [connection, setConnection] = useState("");
  const [panel, setPanel] = useState(false);
  const [saveOpen, setSaveOpen] = useState(false);
  const [workspaces, setWorkspaces] = useState<{ id: string; name: string }[]>(
    [],
  );
  const [workspace, setWorkspace] = useState(params.get("workspace") || "");
  const [saved, setSaved] = useState<{
    draft_id: string;
    workspace_id: string;
  } | null>(null);
  const sending = useRef(false);
  const messageID = useRef<string | null>(null);
  const file = useRef<HTMLInputElement>(null);
  const scrollEnd = useRef<HTMLDivElement>(null);
  const active = session?.operations.find((o) => !terminal(o.state));
  const busy = pending || !!active;
  const artifact = session?.document.artifacts.at(-1);
  const sessionID = session?.id;
  const token = useCallback(async () => {
    try {
      return await getAccessToken();
    } catch {
      return undefined;
    }
  }, [getAccessToken]);

  useEffect(() => {
    let alive = true;
    vibeFetch<VibeConfig>("/config")
      .then((c) => {
        if (alive) {
          setConfig(c);
          if (!params.get("session")) setModels(c.defaults);
        }
      })
      .catch(() => {
        if (alive) setError("Vibe is not connected to the local backend yet.");
      });
    return () => {
      alive = false;
    };
  }, [params]);
  useEffect(() => {
    const id = params.get("session");
    if (!id) return;
    let alive = true;
    void (async () => {
      const auth = await token();
      try {
        let v: Session;
        try {
          v = await vibeFetch<Session>(`/sessions/${id}`, auth);
        } catch (e) {
          if (!auth || !(e instanceof VibeError) || e.code !== "not_found")
            throw e;
          v = await vibeFetch<Session>(`/sessions/${id}/claim`, auth, {
            method: "POST",
            body: "{}",
          });
        }
        if (alive) {
          setSession(v);
          setModels(v.document.models);
        }
      } catch (e) {
        if (alive) setError((e as Error).message);
      }
    })();
    return () => {
      alive = false;
    };
  }, [params, token]);
  useEffect(() => {
    if (!sessionID) return;
    const controller = new AbortController();
    let timer: ReturnType<typeof setTimeout>;
    const connect = async () => {
      try {
        await watchVibe(sessionID, await token(), controller.signal, (v) => {
          setSession((old) =>
            old &&
            old.id === v.id &&
            ((old.event_cursor || 0) > (v.event_cursor || 0) ||
              old.revision > v.revision)
              ? old
              : v,
          );
          setConnection("");
        });
      } catch {
        if (!controller.signal.aborted)
          setConnection("Reconnecting to saved progress…");
      }
      if (!controller.signal.aborted) timer = setTimeout(connect, 2000);
    };
    void connect();
    return () => {
      controller.abort();
      clearTimeout(timer);
    };
  }, [sessionID, token]);
  useEffect(() => {
    scrollEnd.current?.scrollIntoView({ behavior: "smooth", block: "end" });
  }, [session?.document.messages.length, active?.state]);
  const artifactID = artifact?.id;
  useEffect(() => {
    if (artifactID) setPanel(true);
  }, [artifactID]); // open new proposals, preserve user dismissal

  const reload = async (id = sessionID) => {
    if (!id) return;
    const v = await vibeFetch<Session>(`/sessions/${id}`, await token());
    setSession((old) =>
      old && old.id === v.id && (old.event_cursor || 0) > (v.event_cursor || 0)
        ? old
        : v,
    );
    return v;
  };
  async function ensureSession() {
    if (session) return session;
    const id = crypto.randomUUID();
    const v = await vibeFetch<Session>("/sessions", await token(), {
      method: "POST",
      body: JSON.stringify({
        id,
        ...(workspace ? { workspace_id: workspace } : {}),
      }),
    });
    setSession(v);
    window.history.replaceState(
      null,
      "",
      `/vibe-evals?session=${id}${workspace ? `&workspace=${workspace}` : ""}`,
    );
    return v;
  }
  async function submit(
    kind = "message",
    text = content,
    baseline?: Operation,
  ) {
    if (sending.current || busy) return;
    sending.current = true;
    setPending(true);
    setError("");
    try {
      const v = await ensureSession();
      const clientID = messageID.current || crypto.randomUUID();
      messageID.current = clientID;
      await vibeFetch(`/sessions/${v.id}/messages`, await token(), {
        method: "POST",
        body: JSON.stringify({
          client_id: clientID,
          revision: v.revision,
          kind,
          content: text,
          models: baseline
            ? { ...models, evaluator: baseline.models.evaluator }
            : models,
          ...(artifact ? { artifact_id: artifact.id } : {}),
          ...(baseline ? { baseline_id: baseline.id } : {}),
        }),
      });
      messageID.current = null;
      setContent("");
      await reload(v.id);
    } catch (e) {
      setError((e as Error).message);
      if (e instanceof VibeError) {
        messageID.current = null;
        if (e.code === "revision_conflict") await reload();
      }
    } finally {
      sending.current = false;
      setPending(false);
    }
  }
  async function edit(fields: Record<string, unknown>) {
    if (!session) return;
    setPending(true);
    setError("");
    try {
      const v = await vibeFetch<Session>(
        `/sessions/${session.id}`,
        await token(),
        {
          method: "PATCH",
          body: JSON.stringify({ revision: session.revision, ...fields }),
        },
      );
      setSession(v);
    } catch (e) {
      setError((e as Error).message);
      await reload();
    } finally {
      setPending(false);
    }
  }
  async function operationAction(id: string, action: "stop" | "approve") {
    setPending(true);
    setError("");
    try {
      await vibeFetch(`/operations/${id}/${action}`, await token(), {
        method: "POST",
        body: "{}",
      });
      await reload();
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setPending(false);
    }
  }
  async function upload(uploaded?: File) {
    if (!uploaded || busy) return;
    setPending(true);
    setError("");
    try {
      const v = await ensureSession();
      if (uploaded.size > (v.anonymous ? 256 * 1024 : 1024 * 1024))
        throw new Error("This file exceeds the import size limit.");
      const result = await vibeFetch<Session>(
        `/sessions/${v.id}/import`,
        await token(),
        {
          method: "POST",
          headers: {
            "Content-Type": uploaded.name.endsWith(".json")
              ? "application/json"
              : "application/yaml",
            "If-Match": String(v.revision),
          },
          body: new Blob([uploaded]),
        },
      );
      setSession(result);
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setPending(false);
      if (file.current) file.current.value = "";
    }
  }
  async function openSave() {
    setSaveOpen(true);
    const auth = await token();
    if (!auth) return;
    try {
      const me =
        await createApiClient(auth).get<UserMeResponse>("/v1/users/me");
      const available = me.organizations.flatMap((o) =>
        o.workspaces.filter(
          (w) =>
            ["workspace_admin", "workspace_member"].includes(w.role) ||
            o.role === "org_admin",
        ),
      );
      setWorkspaces(available);
      if (!workspace && available[0]) setWorkspace(available[0].id);
    } catch (e) {
      setError((e as Error).message);
    }
  }
  async function save() {
    if (!session || !artifact || !workspace) return;
    setPending(true);
    try {
      const auth = await token();
      await vibeFetch(`/sessions/${session.id}/claim`, auth, {
        method: "POST",
        body: "{}",
      });
      const latest = await reload();
      if (!latest) return;
      const result = await vibeFetch<{
        draft_id: string;
        workspace_id: string;
      }>(`/sessions/${session.id}/save`, auth, {
        method: "POST",
        body: JSON.stringify({
          revision: latest.revision,
          artifact_id: artifact.id,
          workspace_id: workspace,
        }),
      });
      setSaved(result);
      await reload();
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setPending(false);
    }
  }

  return (
    <main className="dark flex h-dvh overflow-hidden bg-background font-sans text-builder-fg">
      <nav
        aria-label="Vibe navigation"
        className="hidden w-56 shrink-0 flex-col border-r border-builder-border bg-sidebar px-4 py-6 md:flex"
      >
        <Link
          href="/"
          className="mb-8 flex items-center gap-2 text-sm font-semibold"
        >
          <ClashMark className="size-5" /> AgentClash
        </Link>
        <Button
          variant="outline"
          className="justify-start"
          onClick={() => {
            window.location.href = "/vibe-evals";
          }}
        >
          <MessageSquarePlus size={15} /> New conversation
        </Button>
        <p className="mt-8 px-2 font-mono text-[10px] uppercase tracking-widest text-builder-fg-subtle">
          Vibe Evals
        </p>
        <p className="mt-3 px-2 text-xs leading-6 text-builder-fg-muted">
          A conversation about making your agent better.
        </p>
        <div className="mt-auto space-y-3 border-t border-builder-border px-2 pt-4 text-xs text-builder-fg-muted">
          <p>Free trial includes a small check and one retest.</p>
          <Link href="/dashboard" className="block hover:text-builder-fg">
            Your workspace ↗
          </Link>
        </div>
      </nav>
      <div className="flex min-w-0 flex-1 flex-col">
        <header className="flex h-16 shrink-0 items-center justify-between border-b border-builder-border px-5">
          <Link href="/vibe-evals" className="text-sm font-medium">
            Vibe Evals{" "}
            <span className="ml-2 rounded border border-builder-border px-1.5 py-0.5 font-mono text-[9px] text-builder-fg-muted">
              PREVIEW
            </span>
          </Link>
          <div className="flex gap-2">
            {workspace && <CreditsDialog workspace={workspace} />}
            <Link
              href="/"
              className="p-2 text-xs text-builder-fg-muted md:hidden"
            >
              AgentClash
            </Link>
            {artifact && (
              <Button
                variant="ghost"
                size="sm"
                onClick={() => setPanel(!panel)}
              >
                <PanelRight size={15} /> Your agent
              </Button>
            )}
          </div>
        </header>
        <div className="min-h-0 flex-1 overflow-y-auto">
          <div className="mx-auto max-w-3xl px-5 py-8 sm:px-8">
            {!session?.document.messages.length && (
              <div className="pb-10 pt-[min(12vh,100px)]">
                <ClashMark className="mb-7 size-8 text-builder-fg-muted" />
                <h1 className="max-w-xl text-3xl font-semibold tracking-tight sm:text-4xl">
                  What are you working on?
                </h1>
                <p className="mt-4 max-w-xl text-sm leading-7 text-builder-fg-muted">
                  Tell me what your agent does, what you want to build, or what
                  isn’t working. We’ll figure out the next step together.
                </p>
                <div className="mt-8 flex flex-wrap gap-2">
                  {starters.map((starter) => (
                    <button
                      key={starter}
                      onClick={() => setContent(starter)}
                      className="rounded-xl border border-builder-border px-3 py-2.5 text-xs text-builder-fg-muted transition-colors hover:bg-builder-surface-hover hover:text-builder-fg"
                    >
                      {starter}
                    </button>
                  ))}
                </div>
              </div>
            )}
            <div className="space-y-8" role="log" aria-label="Conversation">
              {session?.document.messages.map((message) => (
                <div
                  key={message.id}
                  className={
                    message.role === "user"
                      ? "ml-auto max-w-[88%] rounded-2xl bg-builder-fg px-4 py-2 text-background"
                      : "pr-4"
                  }
                >
                  {message.role !== "user" && (
                    <p className="mb-2 flex items-center gap-2 text-xs font-medium">
                      <ClashMark className="size-4" /> AgentClash
                    </p>
                  )}
                  <SafeMarkdown>{message.content}</SafeMarkdown>
                </div>
              ))}
              {session?.operations.map((operation) => (
                <div key={operation.id}>
                  {operation.scorecard && operation.scorecard.total > 0 && (
                    <VibeScorecard
                      operation={operation}
                      baseline={session?.operations.find(
                        (o) => o.id === operation.baseline_id,
                      )}
                      loadEvidence={async (key) =>
                        vibeFetch<CaseResult>(
                          `/operations/${operation.id}/case?key=${encodeURIComponent(key)}`,
                          await token(),
                        )
                      }
                      busy={busy}
                      onImprove={() => {
                        setContent(
                          "Help me improve the accepted agent instructions while keeping the evaluation unchanged. Ask me for any missing policy facts.",
                        );
                      }}
                      onRetest={() => submit("retest", "", operation)}
                    />
                  )}
                  {operation.state === "AWAITING_APPROVAL" && (
                    <div className="rounded-xl border border-builder-border p-5">
                      <h2 className="text-sm font-semibold">
                        Ready when you are
                      </h2>
                      <p className="my-3 text-sm text-builder-fg-muted">
                        This operation will cost at most{" "}
                        {dollars(operation.max_cost_nano_usd)}. We’ll hold that
                        amount and settle the actual provider spend when it
                        finishes.
                      </p>
                      <Button
                        size="sm"
                        disabled={pending}
                        onClick={() => operationAction(operation.id, "approve")}
                      >
                        Run for up to {dollars(operation.max_cost_nano_usd)}
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        disabled={pending}
                        onClick={() => operationAction(operation.id, "stop")}
                      >
                        Dismiss
                      </Button>
                    </div>
                  )}
                  {operation.error && (
                    <p className="mt-3 text-xs leading-5 text-builder-warn">
                      {operation.error.message}
                    </p>
                  )}
                  {operation.state === "CANCELLED" && (
                    <p className="mt-3 text-xs text-builder-fg-muted">
                      Execution: cancelled · Billing:{" "}
                      {operation.billing.toLowerCase()}.{" "}
                      {operation.billing === "RECONCILING" &&
                        "A request already sent to the provider may still be billed; its reservation stays held."}
                    </p>
                  )}
                </div>
              ))}
              {(pending ||
                (active && active.state !== "AWAITING_APPROVAL")) && (
                <div
                  role="status"
                  className="flex items-center gap-3 text-sm text-builder-fg-muted"
                >
                  <Loader2 className="size-4 animate-spin" />
                  {active?.kind === "check" || active?.kind === "retest"
                    ? "Running the examples and checking the evidence…"
                    : "Working on your next step…"}
                  {active && (
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => operationAction(active.id, "stop")}
                      disabled={pending}
                    >
                      <Square size={11} /> Stop
                    </Button>
                  )}
                </div>
              )}
              {saved && (
                <p className="rounded-xl border border-builder-border p-4 text-sm">
                  Your evaluation is saved.{" "}
                  <Link
                    href={`/workspaces/${saved.workspace_id}/challenge-packs/builder/${saved.draft_id}`}
                    className="underline underline-offset-4"
                  >
                    Open it in your workspace
                  </Link>{" "}
                  to expand the tests and run your connected agent. Production
                  monitoring is a separate setup.
                </p>
              )}
            </div>
            <div ref={scrollEnd} />
          </div>
        </div>
        <div className="mx-auto w-full max-w-3xl shrink-0 px-5 pb-5 pt-3 sm:px-8">
          {error && (
            <p
              role="alert"
              className="mb-3 text-xs leading-5 text-builder-warn"
            >
              {error}
            </p>
          )}
          {connection && (
            <p role="status" className="mb-2 text-xs text-builder-fg-muted">
              {connection}
            </p>
          )}
          <form
            onSubmit={(e) => {
              e.preventDefault();
              void submit();
            }}
            className="rounded-2xl border border-builder-border bg-builder-surface p-3 focus-within:border-builder-border-strong"
          >
            <textarea
              aria-label="Message Vibe Evals"
              placeholder="Describe your agent, share an idea, or ask a question…"
              value={content}
              onChange={(e) => {
                setContent(e.target.value);
                messageID.current = null;
              }}
              onKeyDown={(e) => {
                if (
                  e.key === "Enter" &&
                  !e.shiftKey &&
                  !e.nativeEvent.isComposing
                ) {
                  e.preventDefault();
                  if (content.trim()) void submit();
                }
              }}
              className="max-h-48 min-h-16 w-full resize-y bg-transparent px-1 py-1 text-sm leading-6 outline-none placeholder:text-builder-fg-subtle"
              maxLength={session?.anonymous === false ? 65536 : 16384}
            />
            <div className="mt-2 flex items-center justify-between gap-3">
              <div className="flex min-w-0 items-center gap-2">
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  aria-label="Import an evaluation"
                  disabled={busy}
                  onClick={() => file.current?.click()}
                >
                  <Paperclip size={16} />
                </Button>
                <input
                  ref={file}
                  type="file"
                  accept=".json,.yaml,.yml"
                  className="hidden"
                  aria-label="Evaluation file"
                  onChange={(e) => upload(e.target.files?.[0])}
                />
                <ModelSelect
                  label="Assistant"
                  value={models.assistant}
                  models={config?.models || []}
                  onChange={(assistant) => setModels({ ...models, assistant })}
                  disabled={busy}
                />
              </div>
              <Button
                type="submit"
                size="icon"
                aria-label="Send message"
                disabled={busy || !content.trim()}
                className="rounded-full"
              >
                <ArrowUp size={18} />
              </Button>
            </div>
          </form>
          <p className="mt-3 text-center text-[10px] text-builder-fg-subtle">
            Private by default. You review drafts; AgentClash counts the
            results.
          </p>
        </div>
      </div>
      {artifact && panel && (
        <ArtifactPanel
          key={artifact.id}
          artifact={artifact}
          requirements={session?.document.requirements || []}
          models={models}
          choices={config?.models || []}
          anonymous={session?.anonymous ?? true}
          busy={busy}
          onClose={() => setPanel(false)}
          onAccept={() => edit({ artifact_id: artifact.id })}
          onEdit={(agent_prompt) =>
            edit({ artifact_id: artifact.id, agent_prompt })
          }
          onRequirement={(requirement_id, status, statement) =>
            edit({ requirement_id, status, statement })
          }
          onModels={setModels}
          onCheck={() => submit("check", "")}
          onPlay={(text) => submit("playground", text)}
          onSave={openSave}
        />
      )}
      <Dialog open={saveOpen} onOpenChange={setSaveOpen}>
        <DialogContent>
          <DialogTitle>Keep what worked</DialogTitle>
          <DialogDescription>
            Save your agent instructions and editable evaluation. Future checks
            in this conversation use your workspace’s AI credits. Your original
            evidence stays here.
          </DialogDescription>
          {workspaces.length ? (
            <>
              <label className="text-sm">
                Workspace
                <select
                  aria-label="Save workspace"
                  value={workspace}
                  onChange={(e) => setWorkspace(e.target.value)}
                  className="mt-2 w-full rounded-lg border bg-background p-3"
                >
                  {workspaces.map((w) => (
                    <option key={w.id} value={w.id}>
                      {w.name}
                    </option>
                  ))}
                </select>
              </label>
              <Button onClick={save} disabled={pending}>
                {saved ? "Saved" : "Save to workspace"}
              </Button>
              {saved && (
                <Link
                  href={`/workspaces/${saved.workspace_id}/challenge-packs/builder/${saved.draft_id}`}
                  className="text-sm underline"
                >
                  Open your evaluation
                </Link>
              )}
            </>
          ) : (
            <Link
              href={`/auth/login?returnTo=${encodeURIComponent(`/vibe-evals?session=${sessionID || ""}`)}`}
              className="rounded-lg bg-primary px-4 py-3 text-center text-sm text-primary-foreground"
            >
              Sign in to save your work
            </Link>
          )}
        </DialogContent>
      </Dialog>
    </main>
  );
}
