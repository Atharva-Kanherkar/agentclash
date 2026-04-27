import { withAuth } from "@workos-inc/authkit-nextjs";
import { redirect } from "next/navigation";
import { sanitizeReturnTo } from "@/lib/auth/return-to";
import { ClashMark } from "@/components/marketing/clash-mark";
import { LightSpeed } from "./lightspeed";
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
    <main className="grid min-h-screen bg-[#060606] text-white lg:grid-cols-[minmax(0,1fr)_480px]">
      <section className="relative min-h-[42vh] overflow-hidden border-b border-white/10 lg:min-h-screen lg:border-b-0 lg:border-r">
        <LightSpeed intensity={1.2} particleCount={24} quality="medium" />
        <div className="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_42%_45%,transparent_0,rgba(0,0,0,0.22)_42%,rgba(0,0,0,0.72)_100%)]" />
        <div className="absolute bottom-6 left-5 right-5 sm:bottom-8 sm:left-8 lg:bottom-10 lg:left-10">
          <p className="font-mono text-[0.68rem] uppercase tracking-[0.22em] text-white/40">
            Agent evaluation at lightspeed
          </p>
          <h1 className="mt-3 max-w-xl font-[family-name:var(--font-display)] text-4xl leading-[0.98] text-white sm:text-5xl lg:text-6xl">
            Sign in to the arena.
          </h1>
        </div>
      </section>

      <section className="flex min-h-[58vh] items-center justify-center px-5 py-10 lg:min-h-screen lg:px-10">
        <div className="w-full max-w-[380px]">
          <div className="mb-8 flex items-center gap-3">
            <ClashMark className="size-10" />
            <span className="font-mono text-xs uppercase tracking-[0.24em] text-white/45">
              AgentClash
            </span>
          </div>

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
          </div>

          <div className="mt-7 grid gap-3 border-l border-white/12 pl-5 text-sm text-white/48">
            <p>
              <span className="text-white/70">Evaluate:</span> same task, same
              tools.
            </p>
            <p>
              <span className="text-white/70">Replay:</span> every move
              preserved.
            </p>
            <p>
              <span className="text-white/70">Decide:</span> evidence over
              vibes.
            </p>
          </div>
        </div>
      </section>
    </main>
  );
}
