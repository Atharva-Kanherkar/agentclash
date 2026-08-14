import { describe, expect, it } from "vitest";

import { providerAccountCreateFields } from "./provider-account-create-fields";

describe("provider account create fields", () => {
  it("offers custom OpenAI-compatible accounts with a conditional endpoint", () => {
    const providerField = providerAccountCreateFields.find(
      (field) => field.key === "provider_key",
    );
    const baseURLField = providerAccountCreateFields.find(
      (field) => field.key === "base_url",
    );

    expect(providerField?.options).toContainEqual({
      value: "custom",
      label: "Custom / OpenAI-compatible",
    });
    expect(baseURLField).toMatchObject({
      visibleWhen: { key: "provider_key", equals: "custom" },
      requiredWhen: { key: "provider_key", equals: "custom" },
    });
  });
});
