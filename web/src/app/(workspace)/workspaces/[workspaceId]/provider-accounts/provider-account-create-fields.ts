import type { CreateResourceField } from "@/components/infra/create-resource-dialog";

export const providerAccountCreateFields: CreateResourceField[] = [
  {
    key: "provider_key",
    label: "Provider",
    type: "select",
    required: true,
    options: [
      { value: "openai", label: "OpenAI" },
      { value: "anthropic", label: "Anthropic" },
      { value: "gemini", label: "Gemini" },
      { value: "xai", label: "xAI" },
      { value: "openrouter", label: "OpenRouter" },
      { value: "mistral", label: "Mistral" },
      { value: "custom", label: "Custom / OpenAI-compatible" },
    ],
  },
  {
    key: "name",
    label: "Name",
    placeholder: "e.g. Model Gateway",
    required: true,
  },
  {
    key: "api_key",
    label: "API Key",
    placeholder: "Provider API key",
    required: true,
  },
  {
    key: "base_url",
    label: "Base URL",
    placeholder: "https://models.example.com/v1",
    visibleWhen: { key: "provider_key", equals: "custom" },
    requiredWhen: { key: "provider_key", equals: "custom" },
  },
  {
    key: "limits_config",
    label: "Limits Config",
    type: "json",
    placeholder: "{}",
  },
];
