/**
 * Orchestration Detector - Determines if a message needs structured orchestration
 *
 * Simple greetings and basic questions should use simple streaming.
 * Complex investigations need full orchestration with planning/steps/artifacts.
 */

/**
 * Detect if a message needs orchestration (planning/steps/artifacts)
 * or can be handled with simple streaming
 */
export function needsOrchestration(message: string): boolean {
  const lowerMessage = message.toLowerCase().trim();

  const greetings = [
    'hi',
    'hello',
    'hey',
    'good morning',
    'good afternoon',
    'good evening',
    'sup',
    'yo',
    'howdy',
  ];

  if (greetings.some((greeting) => lowerMessage === greeting || lowerMessage.startsWith(greeting + ' '))) {
    return false;
  }

  const metaQuestions = [
    'what can you do',
    'what are your capabilities',
    'how can you help',
    'what do you know',
    'who are you',
    'what is this',
    'help me',
    'how does this work',
  ];

  if (metaQuestions.some((q) => lowerMessage.includes(q))) {
    return false;
  }

  if (lowerMessage.length < 10) {
    return false;
  }

  const dashboardViewQuestions = [
    'what am i seeing',
    'what do i see',
    'what does this show',
    'what is this dashboard',
    'explain this dashboard',
    'what are these panels',
    'what am i looking at',
    'what is this panel',
    'what is the panel',
    'what data',
    'what metrics',
    'what metric',
    'what\'s this showing',
    'what is displayed',
    'what are the trends',
    'what trends',
    'displayed in',
    'showing in',
    'on this dashboard',
    'on this panel',
    'in this panel',
    'in this dashboard',
    'this panel shows',
    'this dashboard shows',
  ];

  if (dashboardViewQuestions.some((q) => lowerMessage.includes(q))) {
    return false;
  }

  const orchestrationKeywords = [
    'why',
    'investigate',
    'troubleshoot',
    'debug',
    'analyze',
    'check',
    'find',
    'error',
    'spike',
    'increase',
    'decrease',
    'problem',
    'issue',
    'failing',
    'not working',
    'slow',
    'high',
    'low',
    'latency',
    'timeout',
    'trace',
    'logs',
    'metrics',
    'dashboard',
  ];

  const hasInvestigationKeyword = orchestrationKeywords.some((keyword) => lowerMessage.includes(keyword));

  if (hasInvestigationKeyword && lowerMessage.length > 20) {
    return true;
  }

  const queryKeywords = [
    'query for',
    'promql',
    'logql',
    'traceql',
    'write a query',
    'create a query',
    'generate a query',
  ];

  if (queryKeywords.some((keyword) => lowerMessage.includes(keyword))) {
    return true;
  }

  if (lowerMessage.length > 30) {
    return true;
  }

  return false;
}
