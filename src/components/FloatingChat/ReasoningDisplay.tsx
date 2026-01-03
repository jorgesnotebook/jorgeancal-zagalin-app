import React, { useState } from 'react';
import { css } from '@emotion/css';
import { GrafanaTheme2 } from '@grafana/data';
import { useStyles2, Icon, Badge } from '@grafana/ui';
import { calculateConfidenceLevel, formatConfidencePercentage, type ReasoningStep, type SourceReference } from '../../types/explainableAI';

interface ReasoningDisplayProps {
  reasoning?: ReasoningStep[];
  sources?: SourceReference[];
  confidence?: number;
  caveats?: string[];
  collapsed?: boolean;
}

export function ReasoningDisplay({
  reasoning,
  sources,
  confidence,
  caveats,
  collapsed = true,
}: ReasoningDisplayProps) {
  const s = useStyles2(getStyles);
  const [isExpanded, setIsExpanded] = useState(!collapsed);

  if (!reasoning || reasoning.length === 0) {
    return null;
  }

  const getStepIcon = (type: string) => {
    switch (type) {
      case 'observation':
        return '🔍';
      case 'hypothesis':
        return '💡';
      case 'analysis':
        return '📊';
      case 'conclusion':
        return '✅';
      case 'verification':
        return '🔬';
      default:
        return '📝';
    }
  };

  const confidenceLevel = confidence !== undefined ? calculateConfidenceLevel(confidence) : null;

  return (
    <div className={s.container}>
      {confidenceLevel && (
        <div className={s.confidenceBadge}>
          <Badge
            text={`Confidence: ${formatConfidencePercentage(confidence!)}`}
            color={confidenceLevel.color}
            icon="info-circle"
          />
        </div>
      )}

      <div className={s.reasoningHeader} onClick={() => setIsExpanded(!isExpanded)}>
        <Icon name={isExpanded ? 'angle-down' : 'angle-right'} />
        <span className={s.headerText}>Reasoning Process ({reasoning.length} steps)</span>
      </div>

      {isExpanded && (
        <div className={s.content}>
          {reasoning.map((step, idx) => (
            <div key={step.id} className={s.step}>
              <div className={s.stepHeader}>
                <span className={s.stepIcon}>{getStepIcon(step.type)}</span>
                <span className={s.stepType}>{step.type}</span>
                {step.confidence !== undefined && (
                  <Badge
                    text={formatConfidencePercentage(step.confidence)}
                    color={calculateConfidenceLevel(step.confidence).color}
                  />
                )}
              </div>
              <div className={s.stepContent}>{step.content}</div>
              {step.sources && step.sources.length > 0 && (
                <div className={s.stepSources}>
                  <Icon name="link" size="sm" />
                  <span>{step.sources.join(', ')}</span>
                </div>
              )}
            </div>
          ))}

          {sources && sources.length > 0 && (
            <div className={s.sourcesSection}>
              <div className={s.sourcesHeader}>
                <Icon name="database" />
                <span>Sources Used</span>
              </div>
              <div className={s.sourcesList}>
                {sources.map((source, idx) => (
                  <div key={idx} className={s.sourceItem}>
                    <span className={s.sourceName}>{source.name}</span>
                    <Badge
                      text={formatConfidencePercentage(source.relevance)}
                      color="blue"
                    />
                  </div>
                ))}
              </div>
            </div>
          )}

          {caveats && caveats.length > 0 && (
            <div className={s.caveatsSection}>
              <div className={s.caveatsHeader}>
                <Icon name="exclamation-triangle" />
                <span>Important Considerations</span>
              </div>
              <ul className={s.caveatsList}>
                {caveats.map((caveat, idx) => (
                  <li key={idx}>{caveat}</li>
                ))}
              </ul>
            </div>
          )}
        </div>
      )}
    </div>
  );
}

const getStyles = (theme: GrafanaTheme2) => ({
  container: css`
    margin-top: ${theme.spacing(2)};
    border: 1px solid ${theme.colors.border.weak};
    border-radius: ${theme.shape.radius.default};
    background: ${theme.colors.background.secondary};
    overflow: hidden;
  `,
  confidenceBadge: css`
    padding: ${theme.spacing(1)};
    border-bottom: 1px solid ${theme.colors.border.weak};
  `,
  reasoningHeader: css`
    display: flex;
    align-items: center;
    gap: ${theme.spacing(1)};
    padding: ${theme.spacing(1.5)};
    cursor: pointer;
    user-select: none;
    font-weight: ${theme.typography.fontWeightMedium};

    &:hover {
      background: ${theme.colors.action.hover};
    }
  `,
  headerText: css`
    flex: 1;
    font-size: ${theme.typography.body.fontSize};
  `,
  content: css`
    border-top: 1px solid ${theme.colors.border.weak};
  `,
  step: css`
    padding: ${theme.spacing(2)};
    border-bottom: 1px solid ${theme.colors.border.weak};

    &:last-child {
      border-bottom: none;
    }
  `,
  stepHeader: css`
    display: flex;
    align-items: center;
    gap: ${theme.spacing(1)};
    margin-bottom: ${theme.spacing(1)};
  `,
  stepIcon: css`
    font-size: 18px;
  `,
  stepType: css`
    flex: 1;
    font-weight: ${theme.typography.fontWeightMedium};
    text-transform: capitalize;
    color: ${theme.colors.text.primary};
  `,
  stepContent: css`
    color: ${theme.colors.text.secondary};
    line-height: 1.5;
    white-space: pre-wrap;
  `,
  stepSources: css`
    display: flex;
    align-items: center;
    gap: ${theme.spacing(0.5)};
    margin-top: ${theme.spacing(1)};
    font-size: ${theme.typography.bodySmall.fontSize};
    color: ${theme.colors.text.disabled};
  `,
  sourcesSection: css`
    padding: ${theme.spacing(2)};
    background: ${theme.colors.background.primary};
    border-top: 1px solid ${theme.colors.border.weak};
  `,
  sourcesHeader: css`
    display: flex;
    align-items: center;
    gap: ${theme.spacing(1)};
    margin-bottom: ${theme.spacing(1)};
    font-weight: ${theme.typography.fontWeightMedium};
    color: ${theme.colors.text.primary};
  `,
  sourcesList: css`
    display: flex;
    flex-direction: column;
    gap: ${theme.spacing(1)};
  `,
  sourceItem: css`
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: ${theme.spacing(0.5)};
  `,
  sourceName: css`
    font-family: ${theme.typography.fontFamilyMonospace};
    font-size: ${theme.typography.bodySmall.fontSize};
    color: ${theme.colors.text.secondary};
  `,
  caveatsSection: css`
    padding: ${theme.spacing(2)};
    background: ${theme.colors.warning.transparent};
    border-top: 1px solid ${theme.colors.border.weak};
  `,
  caveatsHeader: css`
    display: flex;
    align-items: center;
    gap: ${theme.spacing(1)};
    margin-bottom: ${theme.spacing(1)};
    font-weight: ${theme.typography.fontWeightMedium};
    color: ${theme.colors.warning.text};
  `,
  caveatsList: css`
    margin: 0;
    padding-left: ${theme.spacing(3)};
    color: ${theme.colors.text.secondary};
    line-height: 1.5;
  `,
});
