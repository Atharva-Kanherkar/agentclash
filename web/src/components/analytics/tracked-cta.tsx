"use client";

import Link from "next/link";
import type {
  AnchorHTMLAttributes,
  ButtonHTMLAttributes,
  MouseEvent,
} from "react";
import { rememberAcquisitionCTA } from "@/lib/analytics/attribution";
import { captureWebEvent, sanitizePathname } from "@/lib/analytics/posthog-client";
import { WEB_EVENTS } from "@/lib/analytics/events";

export type CTAIntent =
  | "signup"
  | "sign_in"
  | "start_free"
  | "tryout"
  | "demo"
  | "sales";

interface TrackingProps {
  ctaId: string;
  intent: CTAIntent;
  placement: string;
}

type TrackedLinkProps = TrackingProps &
  Omit<AnchorHTMLAttributes<HTMLAnchorElement>, "href"> & {
    href: string;
  };

const ACQUISITION_INTENTS = new Set<CTAIntent>([
  "signup",
  "sign_in",
  "start_free",
]);

function destination(href: string): {
  kind: string;
  path?: string;
} {
  if (href.startsWith("mailto:")) return { kind: "sales" };
  if (href.startsWith("/auth/")) {
    return { kind: "auth", path: sanitizePathname(href.split(/[?#]/, 1)[0]) };
  }
  if (href.startsWith("/try")) {
    return { kind: "tryout", path: sanitizePathname(href.split(/[?#]/, 1)[0]) };
  }
  if (href.startsWith("/")) {
    return { kind: "internal", path: sanitizePathname(href.split(/[?#]/, 1)[0]) };
  }
  try {
    const url = new URL(href);
    if (url.hostname === "cal.com") return { kind: "demo" };
  } catch {
    // The generic external destination below is intentionally URL-free.
  }
  return { kind: "external" };
}

function trackCTA({
  ctaId,
  intent,
  placement,
  href,
}: TrackingProps & { href: string }): void {
  if (ACQUISITION_INTENTS.has(intent)) rememberAcquisitionCTA(ctaId);
  const target = destination(href);
  captureWebEvent(WEB_EVENTS.MARKETING_CTA_CLICKED, {
    cta_id: ctaId,
    intent,
    placement,
    source_path: sanitizePathname(window.location.pathname),
    destination_kind: target.kind,
    ...(target.path ? { destination_path: target.path } : {}),
  });
}

export function TrackedLink({
  ctaId,
  intent,
  placement,
  href,
  onClick,
  ...props
}: TrackedLinkProps) {
  function handleClick(event: MouseEvent<HTMLAnchorElement>) {
    onClick?.(event);
    trackCTA({ ctaId, intent, placement, href });
  }

  return <Link {...props} href={href} onClick={handleClick} />;
}

type TrackedButtonProps = TrackingProps &
  ButtonHTMLAttributes<HTMLButtonElement> & {
    destinationKind?: string;
    destinationPath?: string;
  };

export function TrackedButton({
  ctaId,
  intent,
  placement,
  destinationKind = "action",
  destinationPath,
  onClick,
  ...props
}: TrackedButtonProps) {
  function handleClick(event: MouseEvent<HTMLButtonElement>) {
    onClick?.(event);
    if (ACQUISITION_INTENTS.has(intent)) rememberAcquisitionCTA(ctaId);
    captureWebEvent(WEB_EVENTS.MARKETING_CTA_CLICKED, {
      cta_id: ctaId,
      intent,
      placement,
      source_path: sanitizePathname(window.location.pathname),
      destination_kind: destinationKind,
      ...(destinationPath
        ? { destination_path: sanitizePathname(destinationPath) }
        : {}),
    });
  }

  return <button {...props} onClick={handleClick} />;
}
