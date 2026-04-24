"use client";

import { useAuth } from "@workos-inc/authkit-nextjs/components";
import { toast } from "sonner";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { LogOut, Settings2, Sparkles } from "lucide-react";
import Link from "next/link";
import { restartOnboarding } from "@/components/onboarding/use-onboarding-state";

interface UserMenuProps {
  displayName?: string;
  email?: string;
  avatarUrl?: string;
  orgName?: string;
  orgSlug?: string;
  workspaceId?: string;
}

export function UserMenu({
  displayName,
  email,
  avatarUrl,
  orgName,
  orgSlug,
  workspaceId,
}: UserMenuProps) {
  const { signOut } = useAuth();

  function handleRestartOnboarding() {
    if (!workspaceId) return;
    restartOnboarding(workspaceId);
    toast.success("Onboarding restarted.");
  }
  const initials = (displayName || email || "U")
    .split(" ")
    .map((w) => w[0])
    .join("")
    .slice(0, 2)
    .toUpperCase();

  return (
    <div className="flex items-center gap-3">
      {orgName && (
        <span className="hidden text-[0.6875rem] font-medium text-muted-foreground/50 tracking-wide sm:block">
          {orgName}
        </span>
      )}
      <DropdownMenu>
        <DropdownMenuTrigger className="outline-none rounded-full ring-offset-background focus-visible:ring-2 focus-visible:ring-ring/50 focus-visible:ring-offset-2">
          <Avatar className="size-7">
            {avatarUrl && <AvatarImage src={avatarUrl} alt="" />}
            <AvatarFallback className="bg-white/[0.08] text-[0.5625rem] font-medium text-foreground/70">
              {initials}
            </AvatarFallback>
          </Avatar>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" className="w-52">
          <div className="px-2 py-2">
            <p className="text-sm font-medium">{displayName || "User"}</p>
            {email && (
              <p className="text-xs text-muted-foreground/70 truncate mt-0.5">
                {email}
              </p>
            )}
          </div>
          <DropdownMenuSeparator />
          {orgSlug && (
            <DropdownMenuItem
              render={<Link href={`/orgs/${orgSlug}/members`} />}
            >
              <Settings2 className="size-4" />
              Organization Settings
            </DropdownMenuItem>
          )}
          {workspaceId && (
            <DropdownMenuItem onClick={handleRestartOnboarding}>
              <Sparkles className="size-4" />
              Restart onboarding
            </DropdownMenuItem>
          )}
          <DropdownMenuItem onClick={() => signOut()}>
            <LogOut className="size-4" />
            Sign out
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  );
}
