import { defineConfig } from "@playwright/test";
export default defineConfig({
  testDir: "./e2e/vibe",
  fullyParallel: false,
  workers: 1,
  timeout: 45_000,
  use: {
    baseURL: "http://127.0.0.1:53517",
    trace: "retain-on-failure",
    launchOptions: process.env.VIBE_TEST_CHROMIUM
      ? { executablePath: process.env.VIBE_TEST_CHROMIUM }
      : {},
  },
  webServer: {
    command: "npm run dev -- --hostname 127.0.0.1 --port 53517",
    url: "http://127.0.0.1:53517/vibe-evals",
    reuseExistingServer: true,
    timeout: 120_000,
    env: {
      NEXT_PUBLIC_API_URL: "http://127.0.0.1:55440",
      WORKOS_CLIENT_ID: "client_test_vibe",
      WORKOS_API_KEY: "sk_test_not_a_real_key",
      WORKOS_COOKIE_PASSWORD: "vibe-test-cookie-password-32-characters-long",
      NEXT_PUBLIC_WORKOS_REDIRECT_URI: "http://127.0.0.1:53517/auth/callback",
    },
  },
});
