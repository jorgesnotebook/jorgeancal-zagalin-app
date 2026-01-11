import React from 'react';
import ReactDOM from 'react-dom/client';
import { FloatingChatButton } from './components/FloatingChat/FloatingChatButton';
import { llm } from '@grafana/llm';

/**
 * Check if the current page should show the floating chat button
 */
function shouldShowFloatingChat(): boolean {
  const path = window.location.pathname;

  if (path.startsWith('/login') || path.startsWith('/signup')) {
    return false;
  }

  if (path.startsWith('/admin')) {
    return false;
  }

  if (path.startsWith('/a/jorgeancal-zagalin-app')) {
    return false;
  }

  return true;
}

/**
 * Mount the floating chat button globally
 * This is called once when the plugin is loaded and persists across all pages
 * PERFORMANCE: Mounts UI immediately, checks LLM availability in background
 */
export function mountGlobalChat() {
  if (document.getElementById('zagalin-global-chat')) {
    return;
  }

  const container = document.createElement('div');
  container.id = 'zagalin-global-chat';
  container.style.position = 'fixed';
  container.style.zIndex = '9999';
  container.style.pointerEvents = 'none';

  document.body.appendChild(container);

  const root = ReactDOM.createRoot(container);

  const updateVisibility = () => {
    container.style.display = shouldShowFloatingChat() ? 'block' : 'none';
  };

  root.render(
    <div style={{ pointerEvents: 'auto' }}>
      <FloatingChatButton />
    </div>
  );

  updateVisibility();

  window.addEventListener('popstate', updateVisibility);

  const originalPushState = history.pushState;
  const originalReplaceState = history.replaceState;

  history.pushState = function (...args) {
    originalPushState.apply(this, args);
    updateVisibility();
  };

  history.replaceState = function (...args) {
    originalReplaceState.apply(this, args);
    updateVisibility();
  };

  console.log('Zagalin: Global floating chat mounted (shows on dashboards and explore)');

  setTimeout(async () => {
    try {
      const result = await llm.enabled();
      const isEnabled = typeof result === 'boolean' ? result : (result as any)?.ok;

      if (!isEnabled) {
        console.log('Zagalin: LLM plugin not configured, hiding floating chat');
        container.style.display = 'none';
      } else {
        console.log('Zagalin: LLM plugin available and enabled');
      }
    } catch (err) {
      console.log('Zagalin: LLM plugin not available, hiding floating chat');
      container.style.display = 'none';
    }
  }, 0);
}
