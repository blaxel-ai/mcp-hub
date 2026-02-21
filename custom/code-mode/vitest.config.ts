import { defineConfig } from "vitest/config";

export default defineConfig({
  test: {
    testTimeout: 360_000,
    hookTimeout: 30_000,
    setupFiles: ["dotenv/config"],
  },
});
