import { test, expect } from './fixtures';

test('should be possible to view app configuration', async ({ appConfigPage, page }) => {
  // Check for main configuration sections
  await expect(page.getByText('Zagalin Configuration')).toBeVisible();
  await expect(page.getByText('Personality & Behavior')).toBeVisible();
  await expect(page.getByText('Skills & Features')).toBeVisible();
  await expect(page.getByText('UI Preferences')).toBeVisible();

  // Check for save button
  const saveButtons = await page.getByText('Save Configuration').all();
  expect(saveButtons.length).toBeGreaterThan(0);
});
