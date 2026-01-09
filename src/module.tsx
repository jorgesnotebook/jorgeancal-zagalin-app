import React, { Suspense, lazy, Component, ErrorInfo, ReactNode } from 'react';
import { AppPlugin, type AppRootProps } from '@grafana/data';
import { LoadingPlaceholder, Alert } from '@grafana/ui';
import { mountGlobalChat } from './globalChatMount';

const LazyApp = lazy(() => import('./components/App/App'));
const LazyAppConfig = lazy(() => import('./components/AppConfig/AppConfig').then(m => ({ default: m.AppConfig })));

mountGlobalChat();

const App = (props: AppRootProps) => (
  <Suspense fallback={<LoadingPlaceholder text="" />}>
    <LazyApp {...props} />
  </Suspense>
);

class ConfigErrorBoundary extends Component<
  { children: ReactNode },
  { hasError: boolean; error?: Error }
> {
  constructor(props: { children: ReactNode }) {
    super(props);
    this.state = { hasError: false };
  }

  static getDerivedStateFromError(error: Error) {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    console.error('AppConfig Error:', error, errorInfo);
  }

  render() {
    if (this.state.hasError) {
      return (
        <div style={{ padding: '20px' }}>
          <Alert title="Configuration Page Error" severity="error">
            <p>The configuration page failed to load. This may be due to missing dependencies.</p>
            <p>Error: {this.state.error?.message || 'Unknown error'}</p>
            <p>Please check that the Grafana LLM plugin is installed and configured.</p>
          </Alert>
        </div>
      );
    }

    return this.props.children;
  }
}

const ConfigPage = (props: any) => (
  <ConfigErrorBoundary>
    <Suspense fallback={<LoadingPlaceholder text="Loading configuration..." />}>
      <LazyAppConfig {...props} />
    </Suspense>
  </ConfigErrorBoundary>
);

export const plugin = new AppPlugin<{}>().setRootPage(App).addConfigPage({
  title: 'Configuration',
  icon: 'cog',
  body: ConfigPage,
  id: 'configuration',
});
