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
    <main className="relative min-h-screen overflow-hidden bg-[#060606] text-white">
      <div className="absolute inset-0">
        <LightSpeed intensity={1.2} particleCount={24} quality="medium" />
      </div>
      <div className="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_38%_46%,transparent_0,rgba(0,0,0,0.18)_42%,rgba(0,0,0,0.7)_100%)]" />

      <div className="relative grid min-h-screen grid-rows-[1fr_auto] lg:grid-cols-[minmax(0,1fr)_minmax(380px,440px)] lg:grid-rows-1">
        <div className="pointer-events-none flex flex-col justify-end p-6 sm:p-10 lg:p-14">
          <p className="font-mono text-[0.66rem] uppercase tracking-[0.28em] text-white/45">
            Agent evaluation at lightspeed
          </p>
          <h1 className="mt-3 max-w-xl font-mono text-3xl font-medium uppercase leading-[1.05] tracking-[0.04em] text-white sm:text-4xl lg:text-[2.6rem]">
            Sign in
            <br />
            to the arena.
          </h1>
        </div>

        <aside className="flex items-center justify-center px-5 py-10 sm:px-8 lg:px-10">
          <div className="pointer-events-auto w-full max-w-[380px]">
            <div className="mb-7 flex items-center gap-3">
              <ClashMark className="size-9" />
              <span className="font-mono text-[0.7rem] uppercase tracking-[0.26em] text-white/55">
                AgentClash
              </span>
            </div>

            <div className="glass-card glass-shine rounded-2xl p-7">
              <p className="font-mono text-[0.65rem] uppercase tracking-[0.22em] text-white/55">
                Secure login
              </p>
              <h2 className="mt-3 text-2xl font-semibold text-white">
                Welcome back
              </h2>
              <p className="mt-2 text-sm leading-6 text-white/60">
                Continue to your AgentClash dashboard.
              </p>

              <div className="mt-6">
                <SignInButton returnTo={returnTo} />
              </div>
            </div>

            <div className="mt-6 grid gap-2.5 border-l border-white/15 pl-5 text-sm text-white/55">
              <p>
                <span className="text-white/85">Evaluate:</span> same task, same
                tools.
              </p>
              <p>
                <span className="text-white/85">Replay:</span> every move
                preserved.
              </p>
              <p>
                <span className="text-white/85">Decide:</span> evidence over
                vibes.
              </p>
            </div>
          </div>
        </aside>
      </div>
    </main>
  );
}
