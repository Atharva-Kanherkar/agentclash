export type GlossaryTermKey =
  | "agent"
  | "build"
  | "deployment"
  | "challenge-pack"
  | "input-set"
  | "clash"
  | "comparison-mode"
  | "replay"
  | "score"
  | "regression-coverage"
  | "official-pack-mode";

export interface GlossaryEntry {
  label: string;
  definition: string;
}

export const GLOSSARY: Record<GlossaryTermKey, GlossaryEntry> = {
  agent: {
    label: "Agent",
    definition:
      "A configured AI model that competes on a task. Agents call tools to get things done.",
  },
  build: {
    label: "Build",
    definition:
      "The spec for how an agent behaves — its prompt, tools, and output format.",
  },
  deployment: {
    label: "Deployment",
    definition:
      "A live agent ready to run. It wires a build to a model and a runtime profile.",
  },
  "challenge-pack": {
    label: "Challenge Pack",
    definition:
      "The task your agents will attempt. Bundles inputs, expected outputs, and scoring.",
  },
  "input-set": {
    label: "Input Set",
    definition:
      "One versioned batch of test inputs inside a challenge pack.",
  },
  clash: {
    label: "Clash",
    definition:
      "One head-to-head where each agent solves the same challenge under the same constraints.",
  },
  "comparison-mode": {
    label: "Comparison Mode",
    definition:
      "Two or more agents run the same inputs side by side so the scoreboard can rank them.",
  },
  replay: {
    label: "Replay",
    definition:
      "The step-by-step trace of what each agent did, thought, and called — so you can see why it won or failed.",
  },
  score: {
    label: "Score",
    definition:
      "Completion, speed, token efficiency, and tool strategy combined. Higher is better.",
  },
  "regression-coverage": {
    label: "Regression Coverage",
    definition:
      "Known-tricky cases layered on top of the pack to make sure nothing broke.",
  },
  "official-pack-mode": {
    label: "Official Pack Mode",
    definition:
      "Whether the run includes the full official pack plus regressions, or only the selected regressions.",
  },
};
