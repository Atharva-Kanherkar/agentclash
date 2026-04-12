"use client";

import { useAuth } from "@workos-inc/authkit-nextjs/components";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { LogOut, Settings } from "lucide-react";

interface UserMenuProps {
  displayName?: string;
  email?: string;
  avatarUrl?: string;
  orgName?: string;
}

export function UserMenu({
  displayName,
  email,
  avatarUrl,
  orgName,
}: UserMenuProps) {
  const { signOut } = useAuth();
  const initials = (displayName || email || "U")
    .split(" ")
    .map((w) => w[0])
    .join("")
    .slice(0, 2)
    .toUpperCase();

  return (
    <div className="flex items-center gap-3">
      {orgName && (
        <span className="hidden text-xs text-muted-foreground sm:block">
          {orgName}
        </span>
      )}
      <DropdownMenu>
        <DropdownMenuTrigger className="outline-none">
          <Avatar className="size-7 cursor-pointer">
            {avatarUrl && <AvatarImage src={avatarUrl} alt="" />}
            <AvatarFallback className="text-[0.625rem]">
              {initials}
            </AvatarFallback>
          </Avatar>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" className="w-48">
          <div className="px-2 py-1.5">
            <p className="text-sm font-medium">{displayName || "User"}</p>
            {email && (
              <p className="text-xs text-muted-foreground truncate">{email}</p>
            )}
          </div>
          <DropdownMenuSeparator />
          <DropdownMenuItem disabled>
            <Settings className="size-4" />
            Settings
          </DropdownMenuItem>
          <DropdownMenuSeparator />
          <DropdownMenuItem onClick={() => signOut()}>
            <LogOut className="size-4" />
            Sign out
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  );
}
