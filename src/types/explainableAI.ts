export interface ReasoningStep {
  id: string;
  type: 'observation' | 'hypothesis' | 'analysis' | 'conclusion' | 'verification';
  content: string;
  confidence: number;
  timestamp: Date;
  sources?: string[];
}

export interface SourceReference {
  type: 'dashboard' | 'panel' | 'metric' | 'log' | 'trace' | 'documentation';
  id: string;
  name: string;
  relevance: number;
}

export interface ExplainableResponse {
  answer: string;
  reasoning: ReasoningStep[];
  sources: SourceReference[];
  confidence: number;
  alternativeApproaches?: string[];
  caveats?: string[];
}

export interface ConfidenceLevel {
  score: number;
  label: 'very-low' | 'low' | 'medium' | 'high' | 'very-high';
  color: 'red' | 'orange' | 'purple' | 'green' | 'blue';
}

export function calculateConfidenceLevel(score: number): ConfidenceLevel {
  if (score >= 0.9) {
    return { score, label: 'very-high', color: 'blue' };
  }
  if (score >= 0.75) {
    return { score, label: 'high', color: 'green' };
  }
  if (score >= 0.5) {
    return { score, label: 'medium', color: 'purple' };
  }
  if (score >= 0.3) {
    return { score, label: 'low', color: 'orange' };
  }
  return { score, label: 'very-low', color: 'red' };
}

export function formatConfidencePercentage(confidence: number): string {
  return `${Math.round(confidence * 100)}%`;
}
