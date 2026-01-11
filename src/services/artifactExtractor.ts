import type { Artifact } from './runService';
import { generateArtifactId } from '../utils/idGenerator';
import { ARTIFACT_VALIDATION } from '../utils/constants';

export function extractArtifacts(text: string): Artifact[] {
  return [
    ...extractCodeBlockPromQL(text),
    ...extractCodeBlockLogQL(text),
    ...extractCodeBlockTraceQL(text),
    ...extractInlinePromQL(text),
    ...extractInlineLogQL(text),
    ...extractTraceIDs(text),
  ];
}

function extractCodeBlockPromQL(text: string): Artifact[] {
  const artifacts: Artifact[] = [];
  const pattern = /```(?:promql|prometheus)\s*\n([\s\S]+?)\n```/gi;
  let match;

  while ((match = pattern.exec(text)) !== null) {
    const query = match[1].trim();
    if (query.length > ARTIFACT_VALIDATION.MINIMUM_QUERY_LENGTH) {
      artifacts.push({
        id: generateArtifactId(),
        type: 'query',
        content: query,
        metadata: {
          signal: 'metrics',
          format: 'promql',
        },
        timestamp: new Date().toISOString(),
      });
    }
  }

  return artifacts;
}

function extractCodeBlockLogQL(text: string): Artifact[] {
  const artifacts: Artifact[] = [];
  const pattern = /```(?:logql|loki)\s*\n([\s\S]+?)\n```/gi;
  let match;

  while ((match = pattern.exec(text)) !== null) {
    const query = match[1].trim();
    if (query.includes('=') && query.length > ARTIFACT_VALIDATION.MINIMUM_QUERY_LENGTH) {
      artifacts.push({
        id: generateArtifactId(),
        type: 'query',
        content: query,
        metadata: {
          signal: 'logs',
          format: 'logql',
        },
        timestamp: new Date().toISOString(),
      });
    }
  }

  return artifacts;
}

function extractCodeBlockTraceQL(text: string): Artifact[] {
  const artifacts: Artifact[] = [];
  const pattern = /```(?:traceql|tempo)\s*\n([\s\S]+?)\n```/gi;
  let match;

  while ((match = pattern.exec(text)) !== null) {
    const query = match[1].trim();
    if (query.length > ARTIFACT_VALIDATION.MINIMUM_TRACEQL_LENGTH) {
      artifacts.push({
        id: generateArtifactId(),
        type: 'query',
        content: query,
        metadata: {
          signal: 'traces',
          format: 'traceql',
        },
        timestamp: new Date().toISOString(),
      });
    }
  }

  return artifacts;
}

function extractInlinePromQL(text: string): Artifact[] {
  const artifacts: Artifact[] = [];
  const pattern = /\b(rate|sum|avg|count|histogram_quantile|increase)\([^)]+\)(?:\{[^}]+\})?(?:\[[^\]]+\])?/g;
  let match;

  while ((match = pattern.exec(text)) !== null) {
    const query = match[0].trim();
    if (query.length > ARTIFACT_VALIDATION.MINIMUM_QUERY_LENGTH && !artifacts.some((a) => a.content === query)) {
      artifacts.push({
        id: generateArtifactId(),
        type: 'query',
        content: query,
        metadata: {
          signal: 'metrics',
          format: 'promql',
        },
        timestamp: new Date().toISOString(),
      });
    }
  }

  return artifacts;
}

function extractInlineLogQL(text: string): Artifact[] {
  const artifacts: Artifact[] = [];
  const pattern = /\{[^}]+\}\s*(?:\|[^|\n]+)*/g;
  let match;

  while ((match = pattern.exec(text)) !== null) {
    const query = match[0].trim();
    if (
      query.includes('=') &&
      query.length > ARTIFACT_VALIDATION.MINIMUM_QUERY_LENGTH &&
      !artifacts.some((a) => a.content === query)
    ) {
      artifacts.push({
        id: generateArtifactId(),
        type: 'query',
        content: query,
        metadata: {
          signal: 'logs',
          format: 'logql',
        },
        timestamp: new Date().toISOString(),
      });
    }
  }

  return artifacts;
}

function extractTraceIDs(text: string): Artifact[] {
  const artifacts: Artifact[] = [];
  const pattern = /\b[0-9a-f]{16,32}\b/gi;
  const seenTraceIDs = new Set<string>();
  let match;

  while ((match = pattern.exec(text)) !== null) {
    const traceID = match[0].toLowerCase();
    if (traceID.length >= ARTIFACT_VALIDATION.MINIMUM_TRACE_ID_LENGTH && !seenTraceIDs.has(traceID)) {
      seenTraceIDs.add(traceID);
      artifacts.push({
        id: generateArtifactId(),
        type: 'trace_id',
        content: traceID,
        metadata: {
          signal: 'traces',
        },
        timestamp: new Date().toISOString(),
      });
    }
  }

  return artifacts;
}
