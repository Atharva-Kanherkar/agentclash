"use client";

import { useAuth } from "@workos-inc/authkit-nextjs/components";
import { Button } from "@/components/ui/button";
import { resetPostHog } from "@/lib/analytics/posthog-client";

export function SignOutButton() {
  const { signOut } = useAuth();

  return (
    <Button
      variant="outline"
      size="sm"
      onClick={() => {
        resetPostHog();
        void signOut();
      }}
    >
      Sign out
    </Button>
  );
}
