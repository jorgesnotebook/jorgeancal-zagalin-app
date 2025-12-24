import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import { PluginType } from '@grafana/data';
import AppConfig from './AppConfig';

// Mock the health service to prevent async calls during tests
jest.mock('../../services/llmHealthService', () => ({
  checkZagalinHealth: jest.fn().mockResolvedValue({
    llm: { enabled: true, provider: 'openai', models: ['gpt-4'] },
    vector: { enabled: false },
  }),
}));

describe('Components/AppConfig', () => {
  let props: any;

  beforeEach(() => {
    jest.resetAllMocks();

    props = {
      plugin: {
        meta: {
          id: 'sample-app',
          name: 'Sample App',
          type: PluginType.app,
          enabled: true,
          jsonData: {},
        },
      },
      query: {},
    };
  });

  test('renders the Zagalin Configuration page with main sections', async () => {
    const plugin = { meta: { ...props.plugin.meta, enabled: false } };

    // @ts-ignore - We don't need to provide `addConfigPage()` and `setChannelSupport()` for these tests
    render(<AppConfig plugin={plugin} query={props.query} />);

    // Wait for async health check to complete and component to render
    await waitFor(() => {
      expect(screen.getByText('Zagalin Configuration')).toBeInTheDocument();
    });

    // Check for section titles
    expect(screen.getByText('Personality & Behavior')).toBeInTheDocument();
    expect(screen.getByText('Skills & Features')).toBeInTheDocument();
    expect(screen.getByText('UI Preferences')).toBeInTheDocument();

    // Check for save button
    expect(screen.getAllByText('Save Configuration')[0]).toBeInTheDocument();
  });
});
