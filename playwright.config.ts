import type { PlaywrightTestConfig } from '@playwright/test';

const config: PlaywrightTestConfig = {
  webServer: {
    command: 'npm run dev',
    port: 5173,
    reuseExistingServer: !process.env.CI,
  },
  testDir: 'tests',
  testMatch: /(.+\.)?(test|spec)\.[jt]s/,
  timeout: 30000,
  use: {
    baseURL: 'http://localhost:5173',
  },
};

export default config;
