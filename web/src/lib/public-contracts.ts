export type PublicMachineContract = {
  path: string;
  title: string;
  contentType: string;
};

// Keep discovery links and route allowlists on one small, edge-safe registry.
// The route handlers still validate and serialize their own sources, but a
// contract cannot silently disappear from llms.txt when a filename changes.
export const PUBLIC_MACHINE_CONTRACTS = [
  {
    path: "/openapi.yaml",
    title: "OpenAPI",
    contentType: "application/yaml",
  },
  {
    path: "/cli-schema.json",
    title: "CLI command schema",
    contentType: "application/json",
  },
  {
    path: "/schemas/prompt-eval.schema.json",
    title: "Prompt eval schema",
    contentType: "application/schema+json",
  },
  {
    path: "/schemas/prompt-eval-result.schema.json",
    title: "Prompt eval result schema",
    contentType: "application/schema+json",
  },
  {
    path: "/schemas/voice-artifact-manifest.schema.json",
    title: "Voice artifact manifest schema",
    contentType: "application/schema+json",
  },
  {
    path: "/schemas/voice-live-continuity-report.schema.json",
    title: "Voice live continuity schema",
    contentType: "application/schema+json",
  },
  {
    path: "/schemas/voice-source-separation-report.schema.json",
    title: "Voice source separation schema",
    contentType: "application/schema+json",
  },
  {
    path: "/schemas/voice-video-sync-report.schema.json",
    title: "Voice video sync schema",
    contentType: "application/schema+json",
  },
] as const satisfies readonly PublicMachineContract[];

export const PUBLIC_SCHEMA_FILENAMES = PUBLIC_MACHINE_CONTRACTS.filter(
  (contract) => contract.path.startsWith("/schemas/"),
).map((contract) => contract.path.slice("/schemas/".length));
