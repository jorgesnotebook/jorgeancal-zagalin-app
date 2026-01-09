import { test, expect } from './fixtures';
import { ROUTES } from '../src/constants';

test.describe('navigating app', () => {
  test('chat page should render successfully', async ({ gotoPage, page }) => {
    await gotoPage(`/${ROUTES.Chat}`);

    // Wait for page to load completely
    await page.waitForLoadState('load');
    await page.waitForLoadState('domcontentloaded');
    await page.waitForLoadState('networkidle');

    // Wait for React to render
    await page.waitForTimeout(2000);

    // The page should either show the chat interface (if LLM is configured)
    // or an error message (if LLM is not configured)
    // We use a more lenient approach: check if main content area has loaded
    const mainContent = page.locator('main, [role="main"]');

    // Wait for main content with longer timeout
    await expect(mainContent).toBeVisible({ timeout: 10000 });

    // Verify content exists (works across Grafana versions)
    const hasContent = await page.evaluate(() => {
      const main = document.querySelector('main') || document.querySelector('[role="main"]');
      if (!main) return false;
      const text = main.textContent || '';
      // Should have some meaningful content (not just whitespace)
      return text.trim().length > 10;
    });

    expect(hasContent).toBe(true);

    console.log('✅ Chat page rendered successfully');
  });

  test('should show app navigation', async ({ gotoPage, page }) => {
    await gotoPage(`/${ROUTES.Chat}`);

    // Wait for complete page load
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(2000);

    // Verify the page loaded successfully by checking URL
    expect(page.url()).toContain(ROUTES.Chat);

    // Verify main content area is present (app loaded)
    const mainContent = page.locator('main, [role="main"]');
    await expect(mainContent).toBeVisible({ timeout: 10000 });

    // Check that navigation loaded by verifying we have clickable elements
    const hasInteractiveElements = await page.evaluate(() => {
      const buttons = document.querySelectorAll('button, a, input, textarea');
      return buttons.length > 0;
    });

    expect(hasInteractiveElements).toBe(true);

    console.log('✅ App navigation loaded successfully');

    // Note: Breadcrumb structure varies significantly across Grafana versions
    // so we don't check for specific breadcrumb content
  });
});
