import React from 'react';
import { Route, Routes } from 'react-router-dom';
import { AppRootProps } from '@grafana/data';
import { css } from '@emotion/css';
import { ChatPanel } from '../FloatingChat/ChatPanel';

function App(props: AppRootProps) {
  return (
    <div className={getAppStyles()}>
      <Routes>
        <Route path="*" element={<ChatPanel />} />
      </Routes>
    </div>
  );
}

const getAppStyles = () => css`
  height: calc(100vh - 52px); /* Account for Grafana header */
  width: 100%;
  overflow: hidden;
  display: flex;
  flex-direction: column;
`;

export default App;
