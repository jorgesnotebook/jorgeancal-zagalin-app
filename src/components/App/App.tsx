import React from 'react';
import { Route, Routes } from 'react-router-dom';
import { AppRootProps } from '@grafana/data';

const AssistantChatPage = React.lazy(() => import('../../pages/AssistantChatPage'));

function App(props: AppRootProps) {
  return (
    <Routes>
      {/* Default page - Zagalin Chat */}
      <Route path="*" element={<AssistantChatPage />} />
    </Routes>
  );
}

export default App;
