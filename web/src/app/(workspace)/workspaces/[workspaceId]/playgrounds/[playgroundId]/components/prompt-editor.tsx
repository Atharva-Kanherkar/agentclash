"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Loader2 } from "@/components/ui/nourico-icons";

interface PromptEditorProps {
  name: string;
  promptTemplate: string;
  systemPrompt: string;
  evaluationSpec: unknown;
  onSave: (data: {
    name: string;
    promptTemplate: string;
    systemPrompt: string;
    evaluationSpec: unknown;
  }) => Promise<void>;
  evalSpecBuilder: React.ReactNode;
}

export function PromptEditor({
  name: initialName,
  promptTemplate: initialPromptTemplate,
  systemPrompt: initialSystemPrompt,
  evaluationSpec: _initialEvaluationSpec,
  onSave,
  evalSpecBuilder,
}: PromptEditorProps) {
  const [name, setName] = useState(initialName);
  const [promptTemplate, setPromptTemplate] = useState(initialPromptTemplate);
  const [systemPrompt, setSystemPrompt] = useState(initialSystemPrompt);
  const [saving, setSaving] = useState(false);

  async function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSaving(true);
    try {
      await onSave({ name, promptTemplate, systemPrompt, evaluationSpec: _initialEvaluationSpec });
    } finally {
      setSaving(false);
    }
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-5">
      <div className="grid gap-4 md:grid-cols-2">
        <div className="space-y-1.5">
          <label className="text-sm font-medium">Name</label>
          <Input value={name} onChange={(e) => setName(e.target.value)} />
        </div>
        <div className="space-y-1.5">
          <label className="text-sm font-medium">System Prompt</label>
          <Input
            value={systemPrompt}
            onChange={(e) => setSystemPrompt(e.target.value)}
            placeholder="Optional system instructions"
          />
        </div>
      </div>

      <div className="space-y-1.5">
        <label className="text-sm font-medium">Prompt Template</label>
        <p className="text-xs text-muted-foreground">
          Use {"{{variable}}"} placeholders that map to test case variables.
        </p>
        <textarea
          value={promptTemplate}
          onChange={(e) => setPromptTemplate(e.target.value)}
          spellCheck={false}
          className="min-h-32 w-full rounded-lg border border-input bg-transparent px-3 py-2 text-sm leading-relaxed focus:outline-none focus:ring-2 focus:ring-ring/50 resize-y"
        />
      </div>

      <div className="space-y-1.5">
        <label className="text-sm font-medium">Evaluation</label>
        {evalSpecBuilder}
      </div>

      <div className="flex justify-end">
        <Button type="submit" disabled={saving}>
          {saving && <Loader2 className="mr-2 size-4 animate-spin" />}
          {saving ? "Saving..." : "Save Playground"}
        </Button>
      </div>
    </form>
  );
}
