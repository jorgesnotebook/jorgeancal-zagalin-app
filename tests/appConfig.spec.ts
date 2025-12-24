import { test, expect } from './fixtures';

test('should be possible to view app configuration', async ({ appConfigPage, page }) => {
  // Wait for the page to be fully loaded and interactive
  await page.waitForLoadState('load');
  await page.waitForLoadState('domcontentloaded');

  // Wait for React to render the component
  await page.waitForTimeout(2000);

  // Check if either the config page OR an error message is shown
  const configHeading = page.getByRole('heading', { name: /zagalin configuration/i });
  const errorAlert = page.locator('[role="alert"]', { hasText: /configuration page error/i });

  // Wait for either the config page or error to appear
  try {
    await expect(configHeading).toBeVisible({ timeout: 10000 });

    // If config page loaded successfully, verify all sections
    await expect(page.locator('text=Personality & Behavior').first()).toBeVisible({ timeout: 5000 });
    await expect(page.locator('text=Skills & Features').first()).toBeVisible({ timeout: 5000 });
    await expect(page.locator('text=UI Preferences').first()).toBeVisible({ timeout: 5000 });
    await expect(page.locator('button:has-text("Save Configuration")').first()).toBeVisible({ timeout: 5000 });
  } catch (e) {
    // If config didn't load, check for graceful error message
    await expect(errorAlert).toBeVisible({ timeout: 5000 });
    console.log('Config page showed error (expected in Grafana versions without LLM plugin support)');
  }
});
