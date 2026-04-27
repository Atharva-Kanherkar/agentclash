import { withAuth } from "@workos-inc/authkit-nextjs";
import { redirect } from "next/navigation";
import { sanitizeReturnTo } from "@/lib/auth/return-to";
import { ClashMark } from "@/components/marketing/clash-mark";
import { SignInButton } from "./sign-in-button";
import { Starfield } from "./starfield";
import { TiltCard } from "./tilt-card";

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
      <Starfield count={1500} velocity={5} />
      <div className="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_50%_50%,transparent_0,rgba(0,0,0,0.25)_55%,rgba(0,0,0,0.85)_100%)]" />

      <div className="pointer-events-none relative grid min-h-screen grid-rows-[1fr_auto] lg:grid-cols-[minmax(0,1fr)_minmax(440px,520px)] lg:grid-rows-1">
        <div className="flex flex-col justify-end p-6 sm:p-10 lg:p-14">
          <p className="font-mono text-[0.66rem] uppercase tracking-[0.28em] text-white/45">
            Open evals engine
          </p>
          <h1 className="mt-3 max-w-2xl font-mono text-[1.65rem] font-medium uppercase leading-[1.05] tracking-[0.04em] text-white sm:text-4xl lg:text-[2.6rem]">
            Evals for LLMs
            <br />
            and agents.
          </h1>
          <p className="mt-4 max-w-md text-[0.85rem] leading-6 text-white/55 sm:mt-5 sm:text-sm">
            Run the same task across models. Score on real outcomes. Replay
            every step.
          </p>
        </div>

        <aside className="flex items-center justify-center px-5 py-10 sm:px-8 lg:px-10">
          <div className="pointer-events-auto w-full max-w-[440px] lg:-translate-y-[6vh]">
            <div className="mb-7 flex items-center gap-3">
              <ClashMark className="size-9" />
              <span className="font-mono text-[0.7rem] uppercase tracking-[0.26em] text-white/55">
                AgentClash
              </span>
            </div>

            <TiltCard>
              <div className="glass-card glass-shine rounded-2xl p-6 sm:p-8 lg:p-9">
                <h2 className="text-2xl font-semibold text-white">
                  Welcome back
                </h2>
                <p className="mt-2 text-sm leading-6 text-white/60">
                  Continue to your AgentClash dashboard.
                </p>

                <div className="mt-6">
                  <SignInButton returnTo={returnTo} />
                </div>
              </div>
            </TiltCard>

            <div className="mt-6 grid gap-2.5 border-l border-white/15 pl-5 text-sm text-white/55">
              <p>
                <span className="text-white/85">Evaluate:</span> same task,
                every model.
              </p>
              <p>
                <span className="text-white/85">Replay:</span> every step
                preserved.
              </p>
              <p>
                <span className="text-white/85">Score:</span> outcomes over
                vibes.
              </p>
            </div>
          </div>
        </aside>
      </div>
    </main>
  );
}
