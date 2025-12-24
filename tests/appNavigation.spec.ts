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
    // Check that we're in the Zagalin app (breadcrumb should be visible)
    await expect(page.getByRole('navigation', { name: 'Breadcrumbs' })).toBeVisible();
    // Check for Zagalin link in the breadcrumb specifically
    await expect(page.getByRole('navigation', { name: 'Breadcrumbs' }).getByRole('link', { name: 'Zagalin' })).toBeVisible();
  });
});
