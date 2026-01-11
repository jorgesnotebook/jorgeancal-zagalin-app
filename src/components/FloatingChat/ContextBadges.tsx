import React from 'react';
import { css } from '@emotion/css';
import { GrafanaTheme2 } from '@grafana/data';
import { IconButton, Tooltip, useStyles2 } from '@grafana/ui';
import { ConversationContext } from '../../services/conversationStorage';

interface ContextBadgesProps {
  contexts: ConversationContext[];
  onRemove?: (dashboardUid: string) => void;
  readOnly?: boolean;
}

export function ContextBadges({ contexts, onRemove, readOnly = false }: ContextBadgesProps) {
  const styles = useStyles2(getStyles);

  if (contexts.length === 0) {
    return null;
  }

  return (
    <div className={styles.container}>
      <div className={styles.label}>
        <span>📊 Attached Contexts ({contexts.length})</span>
      </div>
      <div className={styles.badges}>
        {contexts.map((context) => {
          const displayText = context.panelId
            ? `${context.dashboardTitle} → ${context.panelTitle || `Panel ${context.panelId}`}`
            : context.dashboardTitle;

          return (
            <div key={`${context.dashboardUid}-${context.panelId || 'dashboard'}`} className={styles.badgeWrapper}>
              <Tooltip
                content={
                  <div>
                    <div>
                      <strong>Dashboard:</strong> {context.dashboardTitle}
                    </div>
                    {context.panelId && (
                      <div>
                        <strong>Panel:</strong> {context.panelTitle || `Panel ${context.panelId}`}
                      </div>
                    )}
                    {context.timeFrom && context.timeTo && (
                      <div>
                        <strong>Time Range:</strong> {context.timeFrom} → {context.timeTo}
                      </div>
                    )}
                    <div>
                      <strong>Added:</strong> {new Date(context.addedAt).toLocaleString()}
                    </div>
                  </div>
                }
              >
                <div className={styles.badge}>
                  <span className={styles.badgeText}>{displayText}</span>
                  {!readOnly && onRemove && (
                    <IconButton
                      name="times"
                      size="sm"
                      tooltip="Remove context"
                      onClick={() => onRemove(context.dashboardUid)}
                      className={styles.removeButton}
                      variant="secondary"
                    />
                  )}
                </div>
              </Tooltip>
            </div>
          );
        })}
      </div>
    </div>
  );
}

const getStyles = (theme: GrafanaTheme2) => ({
  container: css`
    margin-bottom: ${theme.spacing(2)};
    padding: ${theme.spacing(1.5)};
    background: ${theme.colors.background.secondary};
    border-radius: ${theme.shape.radius.default};
    border: 1px solid ${theme.colors.border.weak};
  `,
  label: css`
    font-size: ${theme.typography.bodySmall.fontSize};
    font-weight: ${theme.typography.fontWeightMedium};
    color: ${theme.colors.text.secondary};
    margin-bottom: ${theme.spacing(1)};
    display: flex;
    align-items: center;
    gap: ${theme.spacing(1)};
  `,
  badges: css`
    display: flex;
    flex-wrap: wrap;
    gap: ${theme.spacing(1)};
  `,
  badgeWrapper: css`
    display: inline-block;
  `,
  badge: css`
    display: inline-flex;
    align-items: center;
    gap: ${theme.spacing(0.5)};
    padding: ${theme.spacing(0.5, 1)};
    background: ${theme.colors.primary.transparent};
    border: 1px solid ${theme.colors.primary.border};
    border-radius: ${theme.shape.radius.default};
    font-size: ${theme.typography.bodySmall.fontSize};
    color: ${theme.colors.primary.text};
    max-width: 300px;
  `,
  badgeText: css`
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  `,
  removeButton: css`
    &&& {
      width: 20px;
      height: 20px;
      min-width: 20px;
      padding: 0;
      color: ${theme.colors.text.secondary};

      &:hover {
        background: ${theme.colors.error.transparent};
        color: ${theme.colors.error.text};
      }
    }
  `,
});
