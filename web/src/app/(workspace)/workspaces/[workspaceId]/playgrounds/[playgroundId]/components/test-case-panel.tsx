"use client";

import { useState } from "react";
import { useConfirm } from "@/components/ui/confirm-dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { JsonField } from "@/components/ui/json-field";
import { EmptyState } from "@/components/ui/empty-state";
import { Loader2, Plus, Trash2, ClipboardList } from "@/components/ui/nourico-icons";
import type { PlaygroundTestCase } from "@/lib/api/types";

function prettyJSON(value: unknown): string {
  return JSON.stringify(value ?? {}, null, 2);
}

interface TestCasePanelProps {
  testCases: PlaygroundTestCase[];
  onCreateTestCase: (data: {
    caseKey: string;
    variables: string;
    expectations: string;
  }) => Promise<void>;
  onUpdateTestCase: (
    testCaseId: string,
    data: { caseKey: string; variables: string; expectations: string },
  ) => Promise<void>;
  onDeleteTestCase: (testCaseId: string) => Promise<void>;
}

export function TestCasePanel({
  testCases,
  onCreateTestCase,
  onUpdateTestCase,
  onDeleteTestCase,
}: TestCasePanelProps) {
  const confirm = useConfirm();
  const [newCaseKey, setNewCaseKey] = useState("");
  const [newVars, setNewVars] = useState('{\n  "topic": "AgentClash"\n}');
  const [newExpect, setNewExpect] = useState(
    '{\n  "expected_output": "AgentClash"\n}',
  );
  const [creating, setCreating] = useState(false);

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault();
    setCreating(true);
    try {
      await onCreateTestCase({
        caseKey: newCaseKey,
        variables: newVars,
        expectations: newExpect,
      });
      setNewCaseKey("");
    } finally {
      setCreating(false);
    }
  }

  async function handleDelete(id: string, key: string) {
    const ok = await confirm({
      title: "Delete test case?",
      description: `This will permanently delete "${key}" and remove its results from future experiments.`,
      confirmLabel: "Delete",
      variant: "danger",
    });
    if (ok) await onDeleteTestCase(id);
  }

  return (
    <div className="space-y-6">
      <form onSubmit={handleCreate} className="rounded-lg border border-border p-4 space-y-4">
        <h3 className="text-sm font-medium">Add Test Case</h3>
        <div className="space-y-1.5">
          <label className="text-xs font-medium text-muted-foreground">Case Key</label>
          <Input
            value={newCaseKey}
            onChange={(e) => setNewCaseKey(e.target.value)}
            placeholder="summary-case-1"
          />
        </div>
        <div className="grid gap-4 md:grid-cols-2">
          <JsonField
            label="Variables"
            value={newVars}
            onChange={setNewVars}
            rows={5}
            description="Input variables for the prompt template"
          />
          <JsonField
            label="Expectations"
            value={newExpect}
            onChange={setNewExpect}
            rows={5}
            description="Ground truth for scoring validators"
          />
        </div>
        <div className="flex justify-end">
          <Button type="submit" disabled={creating} size="sm">
            {creating ? (
              <Loader2 className="mr-2 size-3.5 animate-spin" />
            ) : (
              <Plus className="mr-2 size-3.5" />
            )}
            Add Test Case
          </Button>
        </div>
      </form>

      {testCases.length === 0 ? (
        <EmptyState
          icon={<ClipboardList className="size-10" />}
          title="No test cases yet"
          description="Add test cases with input variables and expected outputs to evaluate your prompt."
        />
      ) : (
        <div className="space-y-3">
          {testCases.map((tc) => (
            <TestCaseCard
              key={tc.id}
              testCase={tc}
              onUpdate={onUpdateTestCase}
              onDelete={() => handleDelete(tc.id, tc.case_key)}
            />
          ))}
        </div>
      )}
    </div>
  );
}

function TestCaseCard({
  testCase,
  onUpdate,
  onDelete,
}: {
  testCase: PlaygroundTestCase;
  onUpdate: TestCasePanelProps["onUpdateTestCase"];
  onDelete: () => void;
}) {
  const [caseKey, setCaseKey] = useState(testCase.case_key);
  const [vars, setVars] = useState(prettyJSON(testCase.variables));
  const [expect, setExpect] = useState(prettyJSON(testCase.expectations));
  const [saving, setSaving] = useState(false);

  async function handleSave(e: React.FormEvent) {
    e.preventDefault();
    setSaving(true);
    try {
      await onUpdate(testCase.id, {
        caseKey,
        variables: vars,
        expectations: expect,
      });
    } finally {
      setSaving(false);
    }
  }

  return (
    <form
      onSubmit={handleSave}
      className="rounded-lg border border-border p-4 space-y-3"
    >
      <div className="flex items-center justify-between gap-3">
        <Input
          value={caseKey}
          onChange={(e) => setCaseKey(e.target.value)}
          className="max-w-xs text-sm font-medium"
        />
        <div className="flex items-center gap-2">
          <Button type="submit" variant="secondary" size="sm" disabled={saving}>
            {saving && <Loader2 className="mr-1.5 size-3 animate-spin" />}
            Save
          </Button>
          <Button
            type="button"
            variant="ghost"
            size="sm"
            onClick={onDelete}
          >
            <Trash2 className="size-3.5 text-muted-foreground" />
          </Button>
        </div>
      </div>
      <div className="grid gap-4 md:grid-cols-2">
        <JsonField label="Variables" value={vars} onChange={setVars} rows={4} />
        <JsonField
          label="Expectations"
          value={expect}
          onChange={setExpect}
          rows={4}
        />
      </div>
    </form>
  );
}
