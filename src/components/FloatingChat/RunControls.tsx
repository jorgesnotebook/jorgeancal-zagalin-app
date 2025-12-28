import React, { useState } from 'react';
import { css } from '@emotion/css';
import { GrafanaTheme2 } from '@grafana/data';
import { useStyles2, IconButton, Modal, Button } from '@grafana/ui';

export type RunStatus = 'pending' | 'planning' | 'executing' | 'paused' | 'completed' | 'cancelled' | 'failed';

interface RunControlsProps {
  status: RunStatus;
  onPause: () => void;
  onResume: () => void;
  onCancel: () => void;
}

export const RunControls: React.FC<RunControlsProps> = ({ status, onPause, onResume, onCancel }) => {
  const styles = useStyles2(getStyles);
  const [showCancelModal, setShowCancelModal] = useState(false);
  const [isLoading, setIsLoading] = useState(false);

  const handlePause = async () => {
    setIsLoading(true);
    try {
      await onPause();
    } finally {
      setIsLoading(false);
    }
  };

  const handleResume = async () => {
    setIsLoading(true);
    try {
      await onResume();
    } finally {
      setIsLoading(false);
    }
  };

  const handleCancelClick = () => {
    setShowCancelModal(true);
  };

  const handleCancelConfirm = async () => {
    setIsLoading(true);
    try {
      await onCancel();
      setShowCancelModal(false);
    } finally {
      setIsLoading(false);
    }
  };

  const handleCancelDismiss = () => {
    setShowCancelModal(false);
  };

  const isRunning = status === 'executing' || status === 'planning' || status === 'paused';

  if (!isRunning) {
    return null;
  }

  return (
    <>
      <div className={styles.container}>
        {status === 'executing' && (
          <IconButton
            name="pause"
            tooltip="Pause execution"
            onClick={handlePause}
            disabled={isLoading}
            size="lg"
            className={styles.pauseButton}
          />
        )}

        {status === 'paused' && (
          <IconButton
            name="play"
            tooltip="Resume execution"
            onClick={handleResume}
            disabled={isLoading}
            size="lg"
            className={styles.resumeButton}
          />
        )}

        <IconButton
          name="times"
          tooltip="Cancel run"
          onClick={handleCancelClick}
          disabled={isLoading}
          size="lg"
          className={styles.cancelButton}
        />
      </div>

      <Modal
        title="Cancel Run"
        isOpen={showCancelModal}
        onDismiss={handleCancelDismiss}
        className={styles.modal}
      >
        <div className={styles.modalContent}>
          <p>Are you sure you want to cancel this run?</p>
          <p className={styles.modalWarning}>This action cannot be undone. All progress will be lost.</p>

          <div className={styles.modalActions}>
            <Button variant="secondary" onClick={handleCancelDismiss} disabled={isLoading}>
              Keep Running
            </Button>
            <Button variant="destructive" onClick={handleCancelConfirm} disabled={isLoading}>
              {isLoading ? 'Cancelling...' : 'Cancel Run'}
            </Button>
          </div>
        </div>
      </Modal>
    </>
  );
};

const getStyles = (theme: GrafanaTheme2) => ({
  container: css`
    display: flex;
    align-items: center;
    justify-content: center;
    gap: ${theme.spacing(1)};
    padding: ${theme.spacing(1.5)};
    background: ${theme.colors.background.secondary};
    border-top: 1px solid ${theme.colors.border.medium};
    position: sticky;
    bottom: 0;
    z-index: 10;
  `,
  pauseButton: css`
    color: #ff6b35;

    &:hover {
      color: #ff8c42;
      background: ${theme.colors.background.canvas};
    }

    &:disabled {
      opacity: 0.5;
      cursor: not-allowed;
    }
  `,
  resumeButton: css`
    color: #ff6b35;

    &:hover {
      color: #ff8c42;
      background: ${theme.colors.background.canvas};
    }

    &:disabled {
      opacity: 0.5;
      cursor: not-allowed;
    }
  `,
  cancelButton: css`
    color: ${theme.colors.error.text};

    &:hover {
      color: ${theme.colors.error.main};
      background: ${theme.colors.error.transparent};
    }

    &:disabled {
      opacity: 0.5;
      cursor: not-allowed;
    }
  `,
  modal: css`
    max-width: 500px;
  `,
  modalContent: css`
    padding: ${theme.spacing(2)};

    p {
      margin-bottom: ${theme.spacing(1.5)};
      color: ${theme.colors.text.primary};
    }
  `,
  modalWarning: css`
    color: ${theme.colors.warning.text};
    font-size: ${theme.typography.bodySmall.fontSize};
    background: ${theme.colors.warning.transparent};
    padding: ${theme.spacing(1)};
    border-radius: ${theme.shape.radius.default};
    border-left: 3px solid ${theme.colors.warning.border};
  `,
  modalActions: css`
    display: flex;
    justify-content: flex-end;
    gap: ${theme.spacing(1)};
    margin-top: ${theme.spacing(2)};
    padding-top: ${theme.spacing(2)};
    border-top: 1px solid ${theme.colors.border.weak};
  `,
});
