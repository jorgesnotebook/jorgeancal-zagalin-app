import React from 'react';
import { MemoryRouter } from 'react-router-dom';
import { AppRootProps, PluginType } from '@grafana/data';
import { render, waitFor } from '@testing-library/react';
import App from './App';

// Mock @grafana/runtime to provide getBackendSrv
jest.mock('@grafana/runtime', () => ({
  ...jest.requireActual('@grafana/runtime'),
  getBackendSrv: jest.fn().mockReturnValue({
    get: jest.fn().mockResolvedValue({
      enabled: true,
      pinned: false,
      jsonData: {},
    }),
    post: jest.fn().mockResolvedValue({}),
  }),
}));

// Mock @grafana/llm - this must be mocked before any imports that use it
jest.mock('@grafana/llm', () => ({
  llm: {
    enabled: jest.fn().mockResolvedValue(true),
    health: jest.fn().mockResolvedValue({
      configured: true,
      enabled: true,
      details: {
        llmProvider: {
          provider: 'openai',
          models: ['gpt-4'],
        },
      },
    }),
    streamChatCompletions: jest.fn().mockImplementation(() => ({
      subscribe: jest.fn(),
    })),
  },
  vector: {
    enabled: jest.fn().mockResolvedValue(false),
  },
}));

describe('Components/App', () => {
  let props: AppRootProps;

  beforeEach(() => {
    jest.resetAllMocks();

    props = {
      basename: 'a/sample-app',
      meta: {
        id: 'sample-app',
        name: 'Sample App',
        type: PluginType.app,
        enabled: true,
        jsonData: {},
      },
      query: {},
      path: '',
      onNavChanged: jest.fn(),
    } as unknown as AppRootProps;
  });

  test('renders without an error"', async () => {
    const { getByPlaceholderText, getByAltText } = render(
      <MemoryRouter>
        <App {...props} />
      </MemoryRouter>
    );

    await waitFor(() => {
      expect(getByPlaceholderText('Ask anything...')).toBeInTheDocument();
      expect(getByAltText('Zagalin')).toBeInTheDocument();
    }, { timeout: 2000 });
  });
});
