import React, { useState, useEffect } from 'react';
import { css } from '@emotion/css';
import { GrafanaTheme2 } from '@grafana/data';
import { useStyles2, IconButton } from '@grafana/ui';
import { ChatPanel } from './ChatPanel';
import { ZagalinColors } from '../../theme/colors';

const STORAGE_KEY = 'zagalin-chat-panel-position';
const SIZE_STORAGE_KEY = 'zagalin-chat-panel-size';

interface Position {
  x: number;
  y: number;
}

interface Size {
  width: number;
  height: number;
}

export function FloatingChatButton() {
  const s = useStyles2(getStyles);
  const [isOpen, setIsOpen] = useState(false);
  const [panelPosition, setPanelPosition] = useState<Position>(() => {
    // Load saved chat panel position from localStorage
    const saved = localStorage.getItem(STORAGE_KEY);
    if (saved) {
      try {
        return JSON.parse(saved);
      } catch (e) {
        // Ignore parse errors
      }
    }
    // Default position for chat panel (bottom-right with some offset)
    return { x: window.innerWidth - 470, y: window.innerHeight - 620 };
  });

  const [panelSize, setPanelSize] = useState<Size>(() => {
    // Load saved chat panel size from localStorage
    const saved = localStorage.getItem(SIZE_STORAGE_KEY);
    if (saved) {
      try {
        return JSON.parse(saved);
      } catch (e) {
        // Ignore parse errors
      }
    }
    // Default size for chat panel
    return { width: 450, height: 600 };
  });

  const [isDragging, setIsDragging] = useState(false);
  const [dragOffset, setDragOffset] = useState({ x: 0, y: 0 });
  const [isResizing, setIsResizing] = useState(false);
  const [resizeStart, setResizeStart] = useState({ x: 0, y: 0, width: 0, height: 0 });

  const toggleChat = () => {
    setIsOpen(!isOpen);
  };

  const handlePanelMouseDown = (e: React.MouseEvent) => {
    // Only start drag from the chat panel header
    const target = e.target as HTMLElement;
    // Allow dragging only if clicking on the header area, not on buttons
    if (target.closest('button')) {
      return;
    }

    const panel = e.currentTarget as HTMLElement;
    const rect = panel.getBoundingClientRect();
    setDragOffset({
      x: e.clientX - rect.left,
      y: e.clientY - rect.top,
    });
    setIsDragging(true);
    e.preventDefault();
  };

  const handleResizeMouseDown = (e: React.MouseEvent) => {
    setResizeStart({
      x: e.clientX,
      y: e.clientY,
      width: panelSize.width,
      height: panelSize.height,
    });
    setIsResizing(true);
    e.preventDefault();
    e.stopPropagation();
  };

  useEffect(() => {
    const handleMouseMove = (e: MouseEvent) => {
      if (isDragging) {
        const newX = e.clientX - dragOffset.x;
        const newY = e.clientY - dragOffset.y;

        // Keep within screen bounds
        const maxX = window.innerWidth - panelSize.width;
        const maxY = window.innerHeight - panelSize.height;

        const boundedX = Math.max(10, Math.min(newX, maxX));
        const boundedY = Math.max(10, Math.min(newY, maxY));

        setPanelPosition({ x: boundedX, y: boundedY });
      }

      if (isResizing) {
        const deltaX = e.clientX - resizeStart.x;
        const deltaY = e.clientY - resizeStart.y;

        const newWidth = Math.max(300, Math.min(resizeStart.width + deltaX, window.innerWidth - 20));
        const newHeight = Math.max(400, Math.min(resizeStart.height + deltaY, window.innerHeight - 20));

        setPanelSize({ width: newWidth, height: newHeight });
      }
    };

    const handleMouseUp = () => {
      if (isDragging) {
        setIsDragging(false);
      }
      if (isResizing) {
        setIsResizing(false);
      }
    };

    if (isDragging || isResizing) {
      document.addEventListener('mousemove', handleMouseMove);
      document.addEventListener('mouseup', handleMouseUp);
    }

    return () => {
      document.removeEventListener('mousemove', handleMouseMove);
      document.removeEventListener('mouseup', handleMouseUp);
    };
  }, [isDragging, dragOffset, isResizing, resizeStart, panelSize.width, panelSize.height]);

  // Save panel position to localStorage when dragging stops
  useEffect(() => {
    if (!isDragging) {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(panelPosition));
    }
  }, [isDragging, panelPosition]);

  // Save panel size to localStorage when resizing stops
  useEffect(() => {
    if (!isResizing) {
      localStorage.setItem(SIZE_STORAGE_KEY, JSON.stringify(panelSize));
    }
  }, [isResizing, panelSize]);

  return (
    <>
      {/* Floating Button - Only show when chat is closed */}
      {!isOpen && (
        <div className={s.floatingButton}>
          <button onClick={toggleChat} className={s.logoButton}>
            <img src="public/plugins/jorgeancal-zagalin-app/img/logo.png" alt="Zagalin" className={s.logoImage} />
          </button>
        </div>
      )}

      {/* Draggable & Resizable Chat Panel */}
      {isOpen && (
        <div
          className={s.chatPanel}
          style={{
            left: `${panelPosition.x}px`,
            top: `${panelPosition.y}px`,
            width: `${panelSize.width}px`,
            height: `${panelSize.height}px`,
            cursor: isDragging ? 'grabbing' : 'default',
          }}
        >
          <div
            className={s.chatPanelHeader}
            onMouseDown={handlePanelMouseDown}
            style={{
              cursor: isDragging ? 'grabbing' : 'grab',
              userSelect: isDragging ? 'none' : 'auto',
            }}
          >
            <h3>Zagalin</h3>
            <IconButton
              name="times"
              size="lg"
              tooltip="Close"
              onClick={toggleChat}
            />
          </div>
          <div className={s.chatPanelContent}>
            <ChatPanel />
          </div>
          {/* Resize Handle */}
          <div
            className={s.resizeHandle}
            onMouseDown={handleResizeMouseDown}
            style={{
              cursor: isResizing ? 'nwse-resize' : 'nwse-resize',
            }}
          />
        </div>
      )}
    </>
  );
}

const getStyles = (theme: GrafanaTheme2) => ({
  floatingButton: css`
    position: fixed;
    bottom: ${theme.spacing(3)};
    right: ${theme.spacing(3)};
    z-index: 1000;
    display: flex;
    flex-direction: column;
    align-items: center;
  `,
  button: css`
    width: 60px;
    height: 60px;
    border-radius: 50%;
    background: ${ZagalinColors.orangeGradient};
    color: white;
    box-shadow: ${theme.shadows.z3};
    transition: all 0.2s ease-in-out;
    border: none;

    &:hover {
      transform: scale(1.1);
      box-shadow: 0 8px 16px rgba(255, 152, 48, 0.3);
      background: ${ZagalinColors.orangeGradientHover};
    }

    svg {
      width: 24px;
      height: 24px;
    }
  `,
  logoButton: css`
    width: 70px;
    height: 70px;
    border-radius: 50%;
    background: ${theme.colors.background.primary};
    border: 2px solid ${theme.colors.border.weak};
    box-shadow: ${theme.shadows.z3};
    transition: all 0.2s ease-in-out;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: ${theme.spacing(1.5)};

    &:hover {
      transform: scale(1.05);
      box-shadow: 0 8px 20px rgba(255, 152, 48, 0.4);
      border-color: ${ZagalinColors.orange};
    }
  `,
  logoImage: css`
    width: 100%;
    height: 100%;
    object-fit: contain;
  `,
  chatPanel: css`
    position: fixed;
    background: ${theme.colors.background.primary};
    border-radius: ${theme.shape.radius.default};
    box-shadow: ${theme.shadows.z3};
    z-index: 999;
    display: flex;
    flex-direction: column;
    animation: slideIn 0.3s ease-out;

    @keyframes slideIn {
      from {
        transform: translateY(20px);
        opacity: 0;
      }
      to {
        transform: translateY(0);
        opacity: 1;
      }
    }

    @media (max-width: 768px) {
      width: calc(100vw - ${theme.spacing(6)}) !important;
      height: calc(100vh - ${theme.spacing(6)}) !important;
    }
  `,
  resizeHandle: css`
    position: absolute;
    right: 0;
    bottom: 0;
    width: 20px;
    height: 20px;
    cursor: nwse-resize;
    background: linear-gradient(135deg, transparent 0%, transparent 50%, ${ZagalinColors.orange} 50%);
    border-bottom-right-radius: ${theme.shape.radius.default};
    opacity: 0.5;
    transition: opacity 0.2s ease;

    &:hover {
      opacity: 1;
    }
  `,
  chatPanelHeader: css`
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: ${theme.spacing(2)};
    border-bottom: 2px solid;
    border-image: ${ZagalinColors.orangeGradient} 1;
    background: linear-gradient(135deg, rgba(242, 204, 12, 0.05) 0%, rgba(255, 152, 48, 0.05) 100%);

    h3 {
      margin: 0;
      font-size: ${theme.typography.h4.fontSize};
      background: ${ZagalinColors.orangeGradient};
      -webkit-background-clip: text;
      -webkit-text-fill-color: transparent;
      background-clip: text;
    }
  `,
  chatPanelContent: css`
    flex: 1;
    overflow: hidden;
  `,
});
