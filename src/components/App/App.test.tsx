import React from 'react';
import { MemoryRouter } from 'react-router-dom';
import { AppRootProps, PluginType } from '@grafana/data';
import { render, waitFor } from '@testing-library/react';
import App from './App';

jest.mock('@grafana/runtime', () => ({
  ...jest.requireActual('@grafana/runtime'),
  getBackendSrv: jest.fn().mockReturnValue({
    get: jest.fn().mockResolvedValue({
      enabled: true,
      pinned: false,
      jsonData: {},
    }),
    post: jest.fn().mockResolvedValue({}),
    delete: jest.fn().mockResolvedValue({}),
    fetch: jest.fn().mockResolvedValue({
      ok: true,
      json: jest.fn().mockResolvedValue({}),
    }),
  }),
  config: {
    buildInfo: {
      version: '9.0.0',
    },
    bootData: {
      user: {
        login: 'test-user',
      },
    },
  },
}));

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
      pipe: jest.fn().mockReturnThis(),
      subscribe: jest.fn(),
    })),
    accumulateContent: jest.fn().mockReturnValue((source: any) => source),
  },
  vector: {
    enabled: jest.fn().mockResolvedValue(false),
  },
}));

jest.mock('../../services/storageApiClient', () => ({
  StorageApiClient: {
    getConversations: jest.fn().mockResolvedValue([]),
    getConversation: jest.fn().mockResolvedValue(null),
    saveConversation: jest.fn().mockResolvedValue({}),
    deleteConversation: jest.fn().mockResolvedValue({}),
    updateTitle: jest.fn().mockResolvedValue({}),
    togglePin: jest.fn().mockResolvedValue({}),
    isAvailable: jest.fn().mockResolvedValue(true),
  },
  migrateFromLocalStorage: jest.fn().mockResolvedValue({ success: true, migrated: 0, errors: 0 }),
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
    const { getByPlaceholderText } = render(
      <MemoryRouter>
        <App {...props} />
      </MemoryRouter>
    );

    await waitFor(() => {
      expect(getByPlaceholderText('Ask anything...')).toBeInTheDocument();
    }, { timeout: 3000 });
  });
});
