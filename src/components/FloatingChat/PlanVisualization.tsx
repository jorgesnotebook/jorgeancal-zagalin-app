import React from 'react';
import { css } from '@emotion/css';
import { GrafanaTheme2 } from '@grafana/data';
import { useStyles2 } from '@grafana/ui';
import { ExecutionPlan } from '../../services/runService';

interface PlanVisualizationProps {
  plan: ExecutionPlan;
  currentStepIndex: number;
}

export const PlanVisualization: React.FC<PlanVisualizationProps> = ({ plan, currentStepIndex }) => {
  const styles = useStyles2(getStyles);

  const completedSteps = plan.steps.filter((s) => s.status === 'completed').length;
  const progressPercent = (completedSteps / plan.steps.length) * 100;

  return (
    <div className={styles.container}>
      <div className={styles.header}>
        <span className={styles.icon}>🎯</span>
        <div className={styles.goalSection}>
          <div className={styles.label}>Goal</div>
          <div className={styles.goal}>{plan.goal}</div>
        </div>
        <div className={styles.duration}>{plan.estimatedDuration}</div>
      </div>

      <div className={styles.progressBar}>
        <div className={styles.progressFill} style={{ width: `${progressPercent}%` }} />
      </div>

      <div className={styles.steps}>
        {plan.steps.map((step, idx) => {
          const isActive = idx === currentStepIndex;
          const isCompleted = step.status === 'completed';
          const isFailed = step.status === 'failed';

          let statusIcon = '⏳';
          if (isCompleted) {
            statusIcon = '✅';
          } else if (isFailed) {
            statusIcon = '❌';
          } else if (isActive) {
            statusIcon = '▶️';
          }

          return (
            <div
              key={idx}
              className={`${styles.step} ${isActive ? styles.stepActive : ''} ${
                isCompleted ? styles.stepCompleted : ''
              } ${isFailed ? styles.stepFailed : ''}`}
            >
              <span className={styles.stepIcon}>{statusIcon}</span>
              <div className={styles.stepContent}>
                <div className={styles.stepTitle}>{step.title}</div>
                <div className={styles.stepDescription}>{step.description}</div>
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
};

const getStyles = (theme: GrafanaTheme2) => ({
  container: css`
    background: ${theme.colors.background.secondary};
    border: 1px solid ${theme.colors.border.medium};
    border-radius: ${theme.shape.radius.default};
    padding: ${theme.spacing(2)};
    margin-bottom: ${theme.spacing(2)};
  `,
  header: css`
    display: flex;
    align-items: flex-start;
    gap: ${theme.spacing(1.5)};
    margin-bottom: ${theme.spacing(2)};
  `,
  icon: css`
    font-size: 24px;
    line-height: 1;
  `,
  goalSection: css`
    flex: 1;
  `,
  label: css`
    font-size: ${theme.typography.bodySmall.fontSize};
    color: ${theme.colors.text.secondary};
    text-transform: uppercase;
    font-weight: ${theme.typography.fontWeightMedium};
    margin-bottom: ${theme.spacing(0.5)};
  `,
  goal: css`
    font-size: ${theme.typography.body.fontSize};
    color: ${theme.colors.text.primary};
    font-weight: ${theme.typography.fontWeightMedium};
  `,
  duration: css`
    font-size: ${theme.typography.bodySmall.fontSize};
    color: ${theme.colors.text.secondary};
    background: ${theme.colors.background.canvas};
    padding: ${theme.spacing(0.5, 1)};
    border-radius: ${theme.shape.radius.default};
  `,
  progressBar: css`
    height: 4px;
    background: ${theme.colors.background.canvas};
    border-radius: 2px;
    overflow: hidden;
    margin-bottom: ${theme.spacing(2)};
  `,
  progressFill: css`
    height: 100%;
    background: linear-gradient(90deg, #ff6b35 0%, #ff8c42 100%);
    transition: width 0.3s ease;
  `,
  steps: css`
    display: flex;
    flex-direction: column;
    gap: ${theme.spacing(1)};
  `,
  step: css`
    display: flex;
    align-items: flex-start;
    gap: ${theme.spacing(1)};
    padding: ${theme.spacing(1)};
    border-radius: ${theme.shape.radius.default};
    transition: all 0.2s ease;
  `,
  stepActive: css`
    background: ${theme.colors.background.canvas};
    animation: pulse 2s ease-in-out infinite;

    @keyframes pulse {
      0%,
      100% {
        opacity: 1;
      }
      50% {
        opacity: 0.8;
      }
    }
  `,
  stepCompleted: css`
    opacity: 0.7;
  `,
  stepFailed: css`
    background: ${theme.colors.error.transparent};
  `,
  stepIcon: css`
    font-size: 16px;
    line-height: 1.5;
  `,
  stepContent: css`
    flex: 1;
    min-width: 0;
  `,
  stepTitle: css`
    font-size: ${theme.typography.body.fontSize};
    color: ${theme.colors.text.primary};
    font-weight: ${theme.typography.fontWeightMedium};
    margin-bottom: ${theme.spacing(0.5)};
  `,
  stepDescription: css`
    font-size: ${theme.typography.bodySmall.fontSize};
    color: ${theme.colors.text.secondary};
    line-height: 1.4;
  `,
});
