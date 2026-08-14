import fs from "node:fs";
import path from "node:path";
import matter from "gray-matter";
import type { DemoMeta } from "./types";

function isDemoMeta(value: unknown): value is DemoMeta {
  if (!value || typeof value !== "object") return false;
  const demo = value as Partial<DemoMeta>;
  return (
    typeof demo.slug === "string" &&
    /^[a-z0-9][a-z0-9-]*$/.test(demo.slug) &&
    typeof demo.name === "string" &&
    typeof demo.sessionMinutes === "number" &&
    Array.isArray(demo.commands) &&
    demo.commands.every(
      (command) =>
        command &&
        typeof command.label === "string" &&
        typeof command.run === "string",
    )
  );
}

function demosDirectory(): string | null {
  for (const candidate of [
    path.join(process.cwd(), "..", "try-cli", "demos"),
    path.join(process.cwd(), "try-cli", "demos"),
  ]) {
    if (fs.existsSync(candidate)) return candidate;
  }
  return null;
}

let cachedDemos: DemoMeta[] | undefined;

export function getBundledTryCliDemos(): DemoMeta[] {
  if (cachedDemos) return [...cachedDemos];
  const directory = demosDirectory();
  if (!directory) return [];

  cachedDemos = fs
    .readdirSync(directory)
    .filter((filename) => filename.endsWith(".trycli.yml"))
    .map((filename) => {
      const source = fs.readFileSync(path.join(directory, filename), "utf8");
      return matter(`---\n${source}\n---`).data as unknown;
    })
    .filter(isDemoMeta)
    .sort((a, b) => a.name.localeCompare(b.name));
  return [...cachedDemos];
}

export function getBundledTryCliDemo(slug: string): DemoMeta | null {
  return getBundledTryCliDemos().find((demo) => demo.slug === slug) ?? null;
}
