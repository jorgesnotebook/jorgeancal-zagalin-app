import { test, expect } from './fixtures';

test('should be possible to view app configuration', async ({ appConfigPage, page }) => {
  // Wait for the page to be fully loaded and interactive
  await page.waitForLoadState('load');
  await page.waitForLoadState('domcontentloaded');

  // Wait for React to render the component
  await page.waitForTimeout(1000);

  // Wait for the main configuration heading to appear
  await expect(page.getByRole('heading', { name: /zagalin configuration/i })).toBeVisible({ timeout: 10000 });

  // Verify main configuration sections are present
  await expect(page.locator('text=Personality & Behavior').first()).toBeVisible({ timeout: 10000 });
  await expect(page.locator('text=Skills & Features').first()).toBeVisible({ timeout: 10000 });
  await expect(page.locator('text=UI Preferences').first()).toBeVisible({ timeout: 10000 });

  // Verify save button exists
  await expect(page.locator('button:has-text("Save Configuration")').first()).toBeVisible({ timeout: 10000 });
});
