import { LogIn, UserPlus, type LucideIcon } from "lucide-react";
import { TrackedLink } from "@/components/analytics/tracked-cta";

/**
 * Single source of truth for the logged-out auth call-to-action. A returning
 * visitor (has the `ac_returning` hint cookie) is sent to sign-in; a likely-new
 * visitor is sent to sign-up. Pure and client-safe (no `next/headers`), so both
 * the client homepage header and the server MarketingHeader can use it.
 */
export function authCta(returning: boolean): {
  href: string;
  label: string;
  Icon: LucideIcon;
} {
  return returning
    ? { href: "/auth/login?mode=signin", label: "Sign in", Icon: LogIn }
    : { href: "/auth/login?mode=signup", label: "Sign up", Icon: UserPlus };
}

/** Compact nav auth CTA shared by the homepage header and MarketingHeader. */
export function AuthCtaLink({
  returning,
  ctaId,
  placement = "nav",
}: {
  returning: boolean;
  ctaId?: string;
  placement?: string;
}) {
  const { href, label, Icon } = authCta(returning);
  const intent = returning ? "sign_in" : "signup";
  return (
    <TrackedLink
      href={href}
      ctaId={ctaId ?? `marketing-header.${placement}.${intent}`}
      intent={intent}
      placement={placement}
      aria-label={label}
      className="inline-flex items-center gap-1.5 rounded-md border border-white/15 bg-white/[0.04] px-2 sm:px-3 py-1.5 lg:px-3.5 lg:py-2 text-white/75 hover:text-white hover:border-white/25 transition-colors"
    >
      <Icon className="size-3.5 lg:size-4" />
      <span className="hidden sm:inline">{label}</span>
    </TrackedLink>
  );
}
