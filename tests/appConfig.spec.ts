import { test, expect } from './fixtures';

test('should be possible to view app configuration', async ({ appConfigPage, page }) => {
  // Wait for the page to be fully loaded and interactive
  await page.waitForLoadState('load');
  await page.waitForLoadState('domcontentloaded');

  // Give extra time for Suspense lazy loading in CI environments
  // Check if the page has any content at all first
  const pageContent = page.locator('body');
  await expect(pageContent).not.toBeEmpty({ timeout: 10000 });

  // Wait for the main configuration heading to appear (indicates lazy loading is complete)
  // Using a more flexible locator that works across Grafana versions
  await expect(page.locator('text=Zagalin Configuration').first()).toBeVisible({ timeout: 30000 });

  // Verify main configuration sections are present
  await expect(page.locator('text=Personality & Behavior').first()).toBeVisible({ timeout: 10000 });
  await expect(page.locator('text=Skills & Features').first()).toBeVisible({ timeout: 10000 });
  await expect(page.locator('text=UI Preferences').first()).toBeVisible({ timeout: 10000 });

  // Verify save button exists
  await expect(page.locator('button:has-text("Save Configuration")').first()).toBeVisible({ timeout: 10000 });
});
