import { test, expect } from './fixtures';

test('should be possible to view app configuration', async ({ appConfigPage, page }) => {
  // Wait for the page to be fully loaded and interactive
  await page.waitForLoadState('load');
  await page.waitForLoadState('domcontentloaded');

  // Wait a bit for React to render
  await page.waitForTimeout(2000);

  // Take screenshot for debugging
  await page.screenshot({ path: 'test-results/config-page-debug.png', fullPage: true });

  // Log the page HTML for debugging
  const bodyHTML = await page.locator('body').innerHTML();
  console.log('Page HTML length:', bodyHTML.length);

  // Check if there are any error messages
  const errorMessages = await page.locator('[role="alert"], .error, .alert-error').allTextContents();
  if (errorMessages.length > 0) {
    console.log('Error messages found:', errorMessages);
  }

  // Check what's actually in the page
  const headings = await page.locator('h1, h2, h3').allTextContents();
  console.log('Headings found:', headings);

  // Wait for the main configuration heading to appear
  // Using a more flexible approach with better error handling
  try {
    await expect(page.locator('h2:has-text("Zagalin Configuration")')).toBeVisible({ timeout: 10000 });
  } catch (e) {
    // If h2 doesn't work, try any heading
    await expect(page.getByRole('heading', { name: /zagalin configuration/i })).toBeVisible({ timeout: 10000 });
  }

  // Verify main configuration sections are present
  await expect(page.locator('text=Personality & Behavior').first()).toBeVisible({ timeout: 10000 });
  await expect(page.locator('text=Skills & Features').first()).toBeVisible({ timeout: 10000 });
  await expect(page.locator('text=UI Preferences').first()).toBeVisible({ timeout: 10000 });

  // Verify save button exists
  await expect(page.locator('button:has-text("Save Configuration")').first()).toBeVisible({ timeout: 10000 });
});
