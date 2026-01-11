/**
 * Context Management Module - Public exports
 */

export { ContextManager } from './ContextManager';
export { useGrafanaContext } from './hooks';
export type {
  GrafanaContext,
  DashboardContext,
  PanelContext,
  UserContext,
  TimeRange,
  TemplateVariable,
  AdhocFilter,
  PanelQuery,
  OptimizedContext,
  PanelData,
  AssistantAction,
  AssistantSkill,
} from './types';
