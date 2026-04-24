"use client";

import type { ReactNode } from "react";
import { HelpCircle } from "lucide-react";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { GLOSSARY, type GlossaryTermKey } from "./glossary";

interface GlossaryTermProps {
  term: GlossaryTermKey;
  children?: ReactNode;
  className?: string;
}

export function GlossaryTerm({ term, children, className }: GlossaryTermProps) {
  const entry = GLOSSARY[term];

  return (
    <span className={className}>
      {children ?? entry.label}
      <TooltipProvider delay={150}>
        <Tooltip>
          <TooltipTrigger
            render={
              <button
                type="button"
                aria-label={`What is ${entry.label}?`}
                className="ml-1 inline-flex size-3.5 translate-y-[-1px] items-center justify-center rounded-full align-middle text-muted-foreground/60 hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/50"
              />
            }
          >
            <HelpCircle className="size-3.5" />
          </TooltipTrigger>
          <TooltipContent side="top" className="max-w-xs leading-relaxed">
            {entry.definition}
          </TooltipContent>
        </Tooltip>
      </TooltipProvider>
    </span>
  );
}
