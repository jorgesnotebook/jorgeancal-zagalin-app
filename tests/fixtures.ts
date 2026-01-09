import { AppConfigPage, AppPage, test as base } from '@grafana/plugin-e2e';
import pluginJson from '../src/plugin.json';

type AppTestFixture = {
  appConfigPage: AppConfigPage;
  gotoPage: (path?: string) => Promise<AppPage>;
  grafanaVersion: string;
};

export const test = base.extend<AppTestFixture>({
  appConfigPage: async ({ gotoAppConfigPage, page }, use) => {
    const configPage = await gotoAppConfigPage({
      pluginId: pluginJson.id,
    });

    // Extra wait for Grafana 12+ (slower loading)
    const version = await getGrafanaVersion(page);
    if (version.startsWith('12.')) {
      await page.waitForTimeout(2000);
    }

    await use(configPage);
  },
  gotoPage: async ({ gotoAppPage, page }, use) => {
    await use(async (path) => {
      const appPage = await gotoAppPage({
        path,
        pluginId: pluginJson.id,
      });

      // Extra wait for Grafana 12+ (slower loading)
      const version = await getGrafanaVersion(page);
      if (version.startsWith('12.')) {
        await page.waitForTimeout(2000);
      }

      return appPage;
    });
  },
  grafanaVersion: async ({ page }, use) => {
    const version = await getGrafanaVersion(page);
    await use(version);
  },
});

async function getGrafanaVersion(page: any): Promise<string> {
  try {
    const version = await page.evaluate(() => {
      // Try to get version from window object
      return (window as any).grafanaBootData?.settings?.buildInfo?.version || 'unknown';
    });
    return version;
  } catch {
    return 'unknown';
  }
}

export { expect } from '@grafana/plugin-e2e';
