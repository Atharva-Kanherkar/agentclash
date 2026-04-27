"use client";

import { ArrowRight } from "lucide-react";
import { signInAction } from "./actions";

interface SignInButtonProps {
  label?: string;
  returnTo?: string;
}

export function SignInButton({
  label = "Continue with AgentClash",
  returnTo = "/dashboard",
}: SignInButtonProps) {
  return (
    <form action={signInAction}>
      <input type="hidden" name="returnTo" value={returnTo} />
      <button
        type="submit"
        className="group flex h-11 w-full items-center justify-center gap-2 rounded-lg border border-white/80 bg-white px-4 text-sm font-semibold text-neutral-950 shadow-[0_20px_60px_rgba(255,255,255,0.14)] transition hover:bg-white/90 focus-visible:outline-none focus-visible:ring-3 focus-visible:ring-white/30 active:translate-y-px"
      >
        <span>{label}</span>
        <ArrowRight
          aria-hidden="true"
          className="size-4 transition-transform group-hover:translate-x-0.5"
        />
      </button>
    </form>
  );
}
