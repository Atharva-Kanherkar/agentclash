import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "Team",
};

const TEAM = [
  { handle: "attharrva15", name: "Atharva" },
  { handle: "PariharCodes", name: "Ayush" },
];

export default function TeamPage() {
  return (
    <main className="min-h-screen flex flex-col items-center justify-center px-6 py-16">
      <h1 className="font-[family-name:var(--font-display)] text-3xl sm:text-4xl text-center tracking-[-0.02em] leading-[1.15]">
        The team behind AgentClash
      </h1>

      <div className="mt-10 flex flex-col gap-4">
        {TEAM.map((member) => (
          <a
            key={member.handle}
            href={`https://x.com/${member.handle}`}
            target="_blank"
            rel="noopener noreferrer"
            className="flex items-center gap-4 rounded-lg border border-white/[0.08] bg-white/[0.03] px-5 py-4 hover:border-white/15 transition-colors"
          >
            <div className="flex-1">
              <p className="text-sm font-medium text-white">
                {member.name}
              </p>
              <p className="text-xs text-white/35">@{member.handle}</p>
            </div>
            <span className="shrink-0 rounded-md bg-white/90 px-3 py-1.5 text-xs font-semibold text-[#060606]">
              Follow
            </span>
          </a>
        ))}
      </div>

      <a
        href="/"
        className="mt-10 text-xs text-white/30 hover:text-white/50 transition-colors"
      >
        &larr; Back to AgentClash
      </a>
    </main>
  );
}
