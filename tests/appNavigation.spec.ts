import { test, expect } from './fixtures';
import { ROUTES } from '../src/constants';

test.describe('navigating app', () => {
  test('chat page should render successfully', async ({ gotoPage, page }) => {
    await gotoPage(`/${ROUTES.Chat}`);
    // Check for Zagalin chat interface elements
    await expect(page.getByPlaceholder('Ask anything...')).toBeVisible();
  });

  test('should show Zagalin logo', async ({ gotoPage, page }) => {
    await gotoPage(`/${ROUTES.Chat}`);
    // Check for Zagalin logo
    await expect(page.getByAltText('Zagalin')).toBeVisible();
  });
});
