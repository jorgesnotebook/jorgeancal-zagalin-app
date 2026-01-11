import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import { PluginType } from '@grafana/data';
import AppConfig from './AppConfig';

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
        addConfigPage: jest.fn(),
        setChannelSupport: jest.fn(),
      },
      query: {},
    };
  });

  test('renders the Zagalin Configuration page with main sections', async () => {
    const plugin = {
      ...props.plugin,
      meta: { ...props.plugin.meta, enabled: false },
    };

    render(<AppConfig plugin={plugin} query={props.query} />);

    await waitFor(() => {
      expect(screen.getByText('Zagalin Configuration')).toBeInTheDocument();
    });

    expect(screen.getByText('LLM Configuration')).toBeInTheDocument();
    expect(screen.getByText('Skills & Features')).toBeInTheDocument();
    expect(screen.getByText('UI Preferences')).toBeInTheDocument();

    expect(screen.getAllByText('Save Configuration')[0]).toBeInTheDocument();
  });
});
