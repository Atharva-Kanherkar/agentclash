import { describe, expect, it } from "vitest";
import {
  providerAccountEndpointHost,
  providerAccountLabel,
} from "./provider-accounts";

describe("provider account labels", () => {
  it("shows only the endpoint host and optional port", () => {
    const account = {
      name: "Compatible",
      provider_key: "custom",
      base_url: "https://models.example.com:8443/private/openai/v1",
    };

    expect(providerAccountEndpointHost(account)).toBe("models.example.com:8443");
    expect(providerAccountLabel(account)).toBe(
      "Compatible (custom · models.example.com:8443)",
    );
    expect(providerAccountLabel(account)).not.toContain("private");
  });

  it("falls back to the provider key for missing or malformed URLs", () => {
    expect(
      providerAccountLabel({
        name: "OpenAI",
        provider_key: "openai",
      }),
    ).toBe("OpenAI (openai)");
    expect(
      providerAccountLabel({
        name: "Broken",
        provider_key: "custom",
        base_url: "not a URL",
      }),
    ).toBe("Broken (custom)");
  });
});
