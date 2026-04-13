"use client";

import { Button } from "@/components/ui/button";
import { ShieldCheck } from "lucide-react";

interface EvaluateReleaseGateDialogProps {
  baselineRunId: string;
  candidateRunId: string;
  onEvaluated: () => void;
}

export function EvaluateReleaseGateDialog(_props: EvaluateReleaseGateDialogProps) {
  // Placeholder — will be implemented in step 3
  return (
    <Button variant="outline" size="sm" disabled>
      <ShieldCheck className="size-4 mr-1.5" />
      Evaluate Release Gate
    </Button>
  );
}
