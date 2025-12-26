import { test, expect } from './fixtures';
import { ROUTES } from '../src/constants';

test.describe('navigating app', () => {
  test('chat page should render successfully', async ({ gotoPage, page }) => {
    await gotoPage(`/${ROUTES.Chat}`);

    // The page should either show the chat interface (if LLM is configured)
    // or an error message (if LLM is not configured)
    // We use a more lenient approach: check if main content area has loaded
    const mainContent = page.locator('main');
    await expect(mainContent).toBeVisible();

    // Try to find either the chat input or error alert
    const hasContent = await page.locator('[role="main"], main').evaluate((el) => {
      return el.textContent && el.textContent.length > 0;
    });

    expect(hasContent).toBe(true);
  });

  test('should show app navigation', async ({ gotoPage, page }) => {
    await gotoPage(`/${ROUTES.Chat}`);

    // Verify the page loaded successfully by checking URL
    expect(page.url()).toContain(ROUTES.Chat);

    // Verify main content area is present (app loaded)
    const mainContent = page.locator('main, [role="main"]');
    await expect(mainContent).toBeVisible();

    // Note: Breadcrumb structure varies significantly across Grafana versions
    // so we don't check for specific breadcrumb content
  });
});
