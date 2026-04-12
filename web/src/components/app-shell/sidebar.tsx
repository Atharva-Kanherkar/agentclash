"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { cn } from "@/lib/utils";
import { navSections } from "./nav-items";
import { PanelLeft } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Sheet,
  SheetContent,
  SheetTitle,
  SheetTrigger,
} from "@/components/ui/sheet";
import { useState } from "react";

interface SidebarProps {
  workspaceId: string;
}

function SidebarContent({ workspaceId }: SidebarProps) {
  const pathname = usePathname();

  return (
    <div className="flex h-full flex-col">
      {/* Logo */}
      <div className="flex h-14 items-center px-4 border-b border-border">
        <Link
          href={`/workspaces/${workspaceId}`}
          className="font-[family-name:var(--font-display)] text-lg text-foreground/90"
        >
          AgentClash
        </Link>
      </div>

      {/* Nav sections */}
      <nav className="flex-1 overflow-y-auto px-3 py-4">
        {navSections.map((section) => (
          <div key={section.title} className="mb-6">
            <p className="mb-1.5 px-2 text-[0.6875rem] font-medium uppercase tracking-wider text-muted-foreground/60">
              {section.title}
            </p>
            {section.items.map((item) => {
              const href = item.href(workspaceId);
              const isActive = pathname.startsWith(href);
              const Icon = item.icon;

              return (
                <Link
                  key={item.label}
                  href={href}
                  className={cn(
                    "flex items-center gap-2.5 rounded-md px-2 py-1.5 text-sm transition-colors",
                    isActive
                      ? "bg-accent text-accent-foreground font-medium"
                      : "text-muted-foreground hover:bg-accent/50 hover:text-foreground",
                  )}
                >
                  <Icon className="size-4 shrink-0" />
                  {item.label}
                </Link>
              );
            })}
          </div>
        ))}
      </nav>
    </div>
  );
}

/** Desktop sidebar — always visible */
export function Sidebar({ workspaceId }: SidebarProps) {
  return (
    <aside className="hidden md:flex md:w-56 md:flex-col md:border-r md:border-border">
      <SidebarContent workspaceId={workspaceId} />
    </aside>
  );
}

/** Mobile sidebar — sheet triggered by hamburger button */
export function MobileSidebar({ workspaceId }: SidebarProps) {
  const [open, setOpen] = useState(false);

  return (
    <Sheet open={open} onOpenChange={setOpen}>
      <SheetTrigger
        render={<Button variant="ghost" size="icon" className="md:hidden" />}
      >
        <PanelLeft className="size-5" />
      </SheetTrigger>
      <SheetContent side="left" className="w-56 p-0">
        <SheetTitle className="sr-only">Navigation</SheetTitle>
        <div onClick={() => setOpen(false)}>
          <SidebarContent workspaceId={workspaceId} />
        </div>
      </SheetContent>
    </Sheet>
  );
}
