import { withAuth } from "@workos-inc/authkit-nextjs";
import { redirect } from "next/navigation";
import { sanitizeReturnTo } from "@/lib/auth/return-to";
import { ClashMark } from "@/components/marketing/clash-mark";
import { SignInButton } from "./sign-in-button";

export default async function LoginPage({
  searchParams,
}: {
  searchParams: Promise<{ returnTo?: string }>;
}) {
  const { returnTo: rawReturnTo } = await searchParams;
  const returnTo = sanitizeReturnTo(rawReturnTo);
  const { user } = await withAuth();
  if (user) redirect(returnTo);

  return (
    <main className="relative flex min-h-screen items-center justify-center overflow-hidden bg-[#060606] px-5 py-12 text-white">
      <div className="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_50%_0%,rgba(255,255,255,0.14),transparent_34%),linear-gradient(180deg,rgba(255,255,255,0.05),transparent_36%)]" />
      <div className="relative grid w-full max-w-5xl gap-10 lg:grid-cols-[1fr_380px] lg:items-center">
        <section className="max-w-2xl">
          <div className="mb-8 flex items-center gap-3">
            <ClashMark className="size-10" />
            <span className="font-mono text-xs uppercase tracking-[0.24em] text-white/45">
              AgentClash
            </span>
          </div>

          <h1 className="font-[family-name:var(--font-display)] text-5xl leading-[0.95] text-white sm:text-6xl">
            Sign in to the arena.
          </h1>
          <p className="mt-5 max-w-xl text-base leading-7 text-white/60">
            Run head-to-head agent evaluations, inspect replays, and keep your
            workspace experiments moving from one secure AgentClash account.
          </p>

          <div className="mt-10 grid max-w-xl gap-3 border-l border-white/12 pl-5 text-sm text-white/48 sm:grid-cols-3 sm:border-l-0 sm:pl-0">
            <div>
              <p className="font-mono text-[0.68rem] uppercase tracking-[0.18em] text-white/35">
                Evaluate
              </p>
              <p className="mt-2 text-white/60">Same task, same tools.</p>
            </div>
            <div>
              <p className="font-mono text-[0.68rem] uppercase tracking-[0.18em] text-white/35">
                Replay
              </p>
              <p className="mt-2 text-white/60">Every move preserved.</p>
            </div>
            <div>
              <p className="font-mono text-[0.68rem] uppercase tracking-[0.18em] text-white/35">
                Decide
              </p>
              <p className="mt-2 text-white/60">Evidence over vibes.</p>
            </div>
          </div>
        </section>

        <div className="rounded-lg border border-white/10 bg-white/[0.045] p-6 shadow-[0_30px_120px_rgba(0,0,0,0.42)] backdrop-blur">
          <div className="mb-6">
            <p className="font-mono text-[0.68rem] uppercase tracking-[0.2em] text-white/38">
              Secure login
            </p>
            <h2 className="mt-3 text-2xl font-semibold text-white">
              Welcome back
            </h2>
            <p className="mt-2 text-sm leading-6 text-white/48">
              Continue to your AgentClash dashboard.
            </p>
          </div>

          <SignInButton returnTo={returnTo} />

          <p className="mt-5 text-center text-xs leading-5 text-white/35">
            Authentication is protected by AgentClash&apos;s configured identity
            provider.
          </p>
        </div>
      </div>
    </main>
  );
}
