import React, { Suspense, lazy } from 'react';
import { AppPlugin, type AppRootProps } from '@grafana/data';
import { LoadingPlaceholder } from '@grafana/ui';
import { mountGlobalChat } from './globalChatMount';
import { AppConfig } from './components/AppConfig/AppConfig';

const LazyApp = lazy(() => import('./components/App/App'));

// Mount the global floating chat when the module loads
// This runs once and persists across all Grafana pages
mountGlobalChat();

const App = (props: AppRootProps) => (
  <Suspense fallback={<LoadingPlaceholder text="" />}>
    <LazyApp {...props} />
  </Suspense>
);

const ConfigPage = (props: any) => <AppConfig {...props} />;

export const plugin = new AppPlugin<{}>().setRootPage(App).addConfigPage({
  title: 'Configuration',
  icon: 'cog',
  body: ConfigPage,
  id: 'configuration',
});
