import Link from "next/link";

export default function NotFound() {
  return (
    <main className="min-h-screen flex flex-col items-center justify-center px-6 py-16 text-center">
      <p className="font-[family-name:var(--font-mono)] text-xs text-white/20 tracking-widest uppercase">
        404
      </p>
      <h1 className="mt-3 font-[family-name:var(--font-display)] text-3xl sm:text-4xl tracking-[-0.02em]">
        This page doesn&apos;t exist yet.
      </h1>
      <p className="mt-4 text-sm text-white/30 max-w-xs leading-relaxed">
        We&apos;re still building. Come back when we&apos;ve shipped more.
      </p>
      <Link
        href="/"
        className="mt-8 text-xs text-white/30 hover:text-white/50 transition-colors"
      >
        &larr; Back to AgentClash
      </Link>
    </main>
  );
}
