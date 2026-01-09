import React, { useState } from 'react';
import { css } from '@emotion/css';
import { GrafanaTheme2 } from '@grafana/data';
import { useStyles2, Icon, IconButton, CodeEditor } from '@grafana/ui';
import { Artifact } from '../../services/runService';

interface ArtifactCardProps {
  artifact: Artifact;
}

export const ArtifactCard: React.FC<ArtifactCardProps> = ({ artifact }) => {
  const styles = useStyles2(getStyles);
  const [isExpanded, setIsExpanded] = useState(false);

  const getIcon = () => {
    switch (artifact.type) {
      case 'query':
        return '📊';
      case 'link':
        return '🔗';
      case 'trace_id':
        return '🔍';
      case 'dashboard_link':
        return '📈';
      case 'tool_call':
        return '🔧';
      default:
        return '📄';
    }
  };

  const getTypeLabel = () => {
    switch (artifact.type) {
      case 'query':
        const signal = artifact.metadata?.signal || 'query';
        return signal.charAt(0).toUpperCase() + signal.slice(1);
      case 'link':
        return 'Link';
      case 'trace_id':
        return 'Trace ID';
      case 'dashboard_link':
        return 'Dashboard';
      case 'tool_call':
        return 'Tool';
      default:
        return artifact.type;
    }
  };

  const getLanguage = () => {
    if (artifact.type === 'query') {
      const format = artifact.metadata?.format || artifact.metadata?.signal;
      if (format === 'promql') {return 'promql';}
      if (format === 'logql') {return 'logql';}
      if (format === 'metrics') {return 'promql';}
      if (format === 'logs') {return 'logql';}
    }
    return 'text';
  };

  const handleCopy = () => {
    navigator.clipboard.writeText(artifact.content);
  };

  const handleOpenExplore = () => {
    const datasourceUid = artifact.metadata?.datasourceUid || artifact.metadata?.datasource;
    if (artifact.type === 'query' && datasourceUid) {
      const query = encodeURIComponent(artifact.content);
      window.open(`/explore?left={"datasource":"${datasourceUid}","queries":[{"expr":"${query}"}]}`, '_blank');
    }
  };

  const canOpenExplore = artifact.type === 'query';

  return (
    <div className={styles.container}>
      <div className={styles.header} onClick={() => setIsExpanded(!isExpanded)}>
        <span className={styles.icon}>{getIcon()}</span>
        <div className={styles.info}>
          <span className={styles.type}>{getTypeLabel()}</span>
          {artifact.metadata?.signal && (
            <span className={styles.signal}>{artifact.metadata.signal}</span>
          )}
        </div>
        <div className={styles.actions}>
          <IconButton
            name="copy"
            tooltip="Copy"
            onClick={(e) => {
              e.stopPropagation();
              handleCopy();
            }}
            size="sm"
          />
          {canOpenExplore && (
            <IconButton
              name="compass"
              tooltip="Open in Explore"
              onClick={(e) => {
                e.stopPropagation();
                handleOpenExplore();
              }}
              size="sm"
            />
          )}
          <Icon name={isExpanded ? 'angle-up' : 'angle-down'} size="lg" />
        </div>
      </div>

      {isExpanded && (
        <div className={styles.content}>
          {artifact.type === 'query' ? (
            <div className={styles.codeEditor}>
              <CodeEditor
                value={artifact.content}
                language={getLanguage()}
                readOnly
                height="100px"
                showLineNumbers={false}
                showMiniMap={false}
              />
            </div>
          ) : (
            <div className={styles.textContent}>{artifact.content}</div>
          )}

          {artifact.metadata && Object.keys(artifact.metadata).length > 0 && (
            <div className={styles.metadata}>
              {Object.entries(artifact.metadata).map(([key, value]) => {
                if (key === 'signal' || key === 'format') {return null;}
                return (
                  <div key={key} className={styles.metadataItem}>
                    <span className={styles.metadataKey}>{key}:</span>
                    <span className={styles.metadataValue}>{String(value)}</span>
                  </div>
                );
              })}
            </div>
          )}
        </div>
      )}
    </div>
  );
};

const getStyles = (theme: GrafanaTheme2) => ({
  container: css`
    background: ${theme.colors.background.secondary};
    border: 1px solid ${theme.colors.border.medium};
    border-left: 3px solid #ff6b35;
    border-radius: ${theme.shape.radius.default};
    margin-bottom: ${theme.spacing(1)};
    overflow: hidden;
  `,
  header: css`
    display: flex;
    align-items: center;
    gap: ${theme.spacing(1)};
    padding: ${theme.spacing(1, 1.5)};
    cursor: pointer;
    user-select: none;

    &:hover {
      background: ${theme.colors.background.canvas};
    }
  `,
  icon: css`
    font-size: 18px;
    line-height: 1;
  `,
  info: css`
    flex: 1;
    display: flex;
    align-items: center;
    gap: ${theme.spacing(1)};
  `,
  type: css`
    font-size: ${theme.typography.bodySmall.fontSize};
    font-weight: ${theme.typography.fontWeightMedium};
    color: ${theme.colors.text.primary};
  `,
  signal: css`
    font-size: ${theme.typography.bodySmall.fontSize};
    color: ${theme.colors.text.secondary};
    background: ${theme.colors.background.canvas};
    padding: ${theme.spacing(0.25, 0.75)};
    border-radius: ${theme.shape.radius.default};
  `,
  actions: css`
    display: flex;
    align-items: center;
    gap: ${theme.spacing(0.5)};
  `,
  content: css`
    padding: ${theme.spacing(1.5)};
    padding-top: 0;
  `,
  codeEditor: css`
    border: 1px solid ${theme.colors.border.weak};
    border-radius: ${theme.shape.radius.default};
    overflow: hidden;
  `,
  textContent: css`
    font-size: ${theme.typography.bodySmall.fontSize};
    color: ${theme.colors.text.primary};
    font-family: ${theme.typography.fontFamilyMonospace};
    white-space: pre-wrap;
    word-break: break-all;
    background: ${theme.colors.background.canvas};
    padding: ${theme.spacing(1)};
    border-radius: ${theme.shape.radius.default};
  `,
  metadata: css`
    margin-top: ${theme.spacing(1)};
    padding-top: ${theme.spacing(1)};
    border-top: 1px solid ${theme.colors.border.weak};
    display: flex;
    flex-direction: column;
    gap: ${theme.spacing(0.5)};
  `,
  metadataItem: css`
    font-size: ${theme.typography.bodySmall.fontSize};
  `,
  metadataKey: css`
    color: ${theme.colors.text.secondary};
    margin-right: ${theme.spacing(0.5)};
  `,
  metadataValue: css`
    color: ${theme.colors.text.primary};
    font-family: ${theme.typography.fontFamilyMonospace};
  `,
});
