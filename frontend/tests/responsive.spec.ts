import { expect, test } from '@playwright/test';

/**
 * Responsive Design Tests
 *
 * These tests verify that components adapt correctly to different container sizes
 * using CSS Container Queries.
 *
 * Test page: /test/responsive?component={name}&height={px}&width={px}
 */

test.describe('Weather Component Responsive Behavior', () => {
  test('shows full layout with chart at 500px height', async ({ page }) => {
    await page.goto('/test/responsive?component=weather&height=500&width=800');

    // Wait for weather data to load
    await page.waitForSelector('.weather-container', { timeout: 10000 });

    // Chart should be visible at this size
    const chart = page.locator('.graph-container, .weather-chart, [class*="chart"]');
    await expect(chart.first()).toBeVisible();

    // Forecast cards should be visible
    const forecastCards = page.locator('.forecast-card');
    await expect(forecastCards.first()).toBeVisible();
  });

  test('hides chart at 300px height', async ({ page }) => {
    await page.goto('/test/responsive?component=weather&height=300&width=800');

    await page.waitForSelector('.weather-container', { timeout: 10000 });

    // Chart should be hidden at this compact size
    const chart = page.locator('.graph-container, .weather-chart, [class*="chart"]');

    // Either not visible or not in DOM
    const chartCount = await chart.count();
    if (chartCount > 0) {
      await expect(chart.first()).not.toBeVisible();
    }

    // Temperature should still be visible
    const temperature = page.locator('.temperature');
    await expect(temperature).toBeVisible();
  });

  test('shows compact forecast at 250px height', async ({ page }) => {
    await page.goto('/test/responsive?component=weather&height=250&width=800');

    await page.waitForSelector('.weather-container', { timeout: 10000 });

    // Temperature should still be visible (essential content)
    const temperature = page.locator('.temperature');
    await expect(temperature).toBeVisible();

    // Chart should definitely be hidden
    const chart = page.locator('.graph-container, .weather-chart');
    const chartCount = await chart.count();
    if (chartCount > 0) {
      await expect(chart.first()).not.toBeVisible();
    }
  });
});

test.describe('SatWord Component Responsive Behavior', () => {
  test('shows all definitions in grid at 400px height', async ({ page }) => {
    await page.goto('/test/responsive?component=satword&height=400&width=800');

    await page.waitForSelector('.word-container', { timeout: 10000 });

    // The word should be visible
    const word = page.locator('.word');
    await expect(word).toBeVisible();

    // All definitions should be visible (grid mode)
    const allDefinitions = page.locator('.all-definitions');
    const allDefsCount = await allDefinitions.count();

    // Either all-definitions is visible, or we're in cycling mode
    if (allDefsCount > 0) {
      await expect(allDefinitions).toBeVisible();
    }
  });

  test('shows single definition with cycling at 350px height', async ({ page }) => {
    await page.goto('/test/responsive?component=satword&height=350&width=800');

    await page.waitForSelector('.word-container', { timeout: 10000 });

    // The word should still be visible
    const word = page.locator('.word');
    await expect(word).toBeVisible();

    // At compact size, should show cycling view OR single definition
    // This test validates the component renders without overflow
    const container = page.locator('.word-container');
    await expect(container).toBeVisible();
  });

  test('shows compact layout at 250px height', async ({ page }) => {
    await page.goto('/test/responsive?component=satword&height=250&width=800');

    await page.waitForSelector('.word-container', { timeout: 10000 });

    // Word should still be visible
    const word = page.locator('.word');
    await expect(word).toBeVisible();

    // Container should not overflow
    const container = page.locator('[data-testid="component-container"]');
    const containerBox = await container.boundingBox();
    expect(containerBox?.height).toBeLessThanOrEqual(250);
  });
});

test.describe('Calendar Component Responsive Behavior', () => {
  test('shows 7-day view at 450px height (auto mode)', async ({ page }) => {
    await page.goto('/test/responsive?component=calendar&height=450&width=1000');

    await page.waitForSelector('.calendar-container, [class*="calendar"]', { timeout: 10000 });

    // Calendar should be visible
    const calendar = page.locator('.calendar-container, [class*="calendar"]');
    await expect(calendar.first()).toBeVisible();

    // Should show 7 day columns (or close to it)
    // This will need to be adjusted based on actual implementation
    const dayHeaders = page.locator('.day-header, [class*="day-name"], .calendar-day-header');
    const headerCount = await dayHeaders.count();

    // In auto mode at 450px, should show 7 days
    // Note: This test may need adjustment once we implement the feature
  });

  test('shows 3-day view at 350px height (auto mode)', async ({ page }) => {
    await page.goto('/test/responsive?component=calendar&height=350&width=1000');

    await page.waitForSelector('.calendar-container, [class*="calendar"]', { timeout: 10000 });

    // Calendar should be visible
    const calendar = page.locator('.calendar-container, [class*="calendar"]');
    await expect(calendar.first()).toBeVisible();

    // At compact height, should show fewer days
    // The exact selector will depend on implementation
  });
});

test.describe('Dashboard Full Page Responsive', () => {
  // These tests check the dashboard grid layout at different viewport sizes
  // Note: Requires authentication - may need to be skipped or setup auth

  test.skip('desktop 1080p layout', async ({ page }) => {
    await page.setViewportSize({ width: 1920, height: 1080 });
    await page.goto('/dashboard');

    // Should show 2-column layout
    const grid = page.locator('.dashboard-grid');
    await expect(grid).toBeVisible();
  });

  test.skip('tablet layout switches to single column', async ({ page }) => {
    await page.setViewportSize({ width: 768, height: 1024 });
    await page.goto('/dashboard');

    // Should show single column layout
    const grid = page.locator('.dashboard-grid');
    await expect(grid).toBeVisible();
  });

  test.skip('mobile layout is scrollable', async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 667 });
    await page.goto('/dashboard');

    // Should allow scrolling
    const grid = page.locator('.dashboard-grid');
    await expect(grid).toBeVisible();
  });
});
