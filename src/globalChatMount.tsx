import React from 'react';
import ReactDOM from 'react-dom/client';
import { FloatingChatButton } from './components/FloatingChat/FloatingChatButton';
import { llm } from '@grafana/llm';

/**
 * Check if the current page should show the floating chat button
 */
function shouldShowFloatingChat(): boolean {
  const path = window.location.pathname;

  // Hide on login/signup pages
  if (path.startsWith('/login') || path.startsWith('/signup')) {
    return false;
  }

  // Hide on admin pages to avoid clutter
  if (path.startsWith('/admin')) {
    return false;
  }

  // Hide on the Zagalin plugin's own pages (already have chat interface there)
  if (path.startsWith('/a/jorgeancal-zagalin-app')) {
    return false;
  }

  // Show on all other pages (dashboards, explore, home, etc.)
  return true;
}

/**
 * Mount the floating chat button globally
 * This is called once when the plugin is loaded and persists across all pages
 */
export async function mountGlobalChat() {
  // Check if already mounted
  if (document.getElementById('zagalin-global-chat')) {
    return;
  }

  // Check if LLM plugin is enabled and configured
  try {
    const result = await llm.enabled();
    const isEnabled = typeof result === 'boolean' ? result : (result as any)?.ok;

    if (!isEnabled) {
      console.log('Zagalin: LLM plugin not configured, skipping floating chat mount');
      return;
    }
  } catch (err) {
    console.log('Zagalin: LLM plugin not available, skipping floating chat mount');
    return;
  }

  // Create container
  const container = document.createElement('div');
  container.id = 'zagalin-global-chat';
  container.style.position = 'fixed';
  container.style.zIndex = '9999';
  container.style.pointerEvents = 'none';

  // Append to body
  document.body.appendChild(container);

  // Create React root and render
  const root = ReactDOM.createRoot(container);

  // Function to update visibility
  const updateVisibility = () => {
    container.style.display = shouldShowFloatingChat() ? 'block' : 'none';
  };

  root.render(
    <div style={{ pointerEvents: 'auto' }}>
      <FloatingChatButton />
    </div>
  );

  // Set initial visibility
  updateVisibility();

  // Listen for route changes (Grafana uses history API)
  window.addEventListener('popstate', updateVisibility);

  // Also observe URL changes via pushState/replaceState
  const originalPushState = history.pushState;
  const originalReplaceState = history.replaceState;

  history.pushState = function(...args) {
    originalPushState.apply(this, args);
    updateVisibility();
  };

  history.replaceState = function(...args) {
    originalReplaceState.apply(this, args);
    updateVisibility();
  };

  console.log('Zagalin: Global floating chat mounted (shows on dashboards and explore)');
}
