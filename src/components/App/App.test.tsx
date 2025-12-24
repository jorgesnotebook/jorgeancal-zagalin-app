import React from 'react';
import { MemoryRouter } from 'react-router-dom';
import { AppRootProps, PluginType } from '@grafana/data';
import { render, waitFor } from '@testing-library/react';
import App from './App';

// Mock @grafana/experimental
jest.mock('@grafana/experimental', () => ({
  llms: {
    openai: {
      enabled: jest.fn().mockResolvedValue(true),
      streamChatCompletions: jest.fn(),
      accumulateContent: jest.fn(),
    },
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
