import { ReasoningStep, SourceReference, ExplainableResponse } from '../types/explainableAI';

const SECTION_PATTERNS = {
  observation: /##\s*🔍\s*Observation([\s\S]*?)(?=##|$)/i,
  analysis: /##\s*📊\s*Analysis([\s\S]*?)(?=##|$)/i,
  hypothesis: /##\s*💡\s*Hypothesis([\s\S]*?)(?=##|$)/i,
  conclusion: /##\s*✅\s*Conclusion([\s\S]*?)(?=##|$)/i,
  verification: /##\s*🔬\s*Verification([\s\S]*?)(?=##|$)/i,
};

const CONFIDENCE_PATTERN = /confidence[:\s]+(\d+)%/gi;
const METRIC_PATTERN = /`([a-z_][a-z0-9_]*)`/gi;

export function parseReasoningResponse(markdownResponse: string): ExplainableResponse | null {
  try {
    const reasoning: ReasoningStep[] = [];
    let overallConfidence = 0.7;

    const obsMatch = markdownResponse.match(SECTION_PATTERNS.observation);
    if (obsMatch) {
      const content = obsMatch[1].trim();
      const confMatch = content.match(CONFIDENCE_PATTERN);
      const confidence = confMatch ? parseInt(confMatch[0].match(/\d+/)?.[0] || '90', 10) / 100 : 0.9;

      reasoning.push({
        id: 'obs-1',
        type: 'observation',
        content,
        confidence,
        timestamp: new Date(),
        sources: extractSources(content),
      });
    }

    const analysisMatch = markdownResponse.match(SECTION_PATTERNS.analysis);
    if (analysisMatch) {
      reasoning.push({
        id: 'analysis-1',
        type: 'analysis',
        content: analysisMatch[1].trim(),
        confidence: 0.85,
        timestamp: new Date(),
      });
    }

    const hypMatch = markdownResponse.match(SECTION_PATTERNS.hypothesis);
    if (hypMatch) {
      reasoning.push({
        id: 'hyp-1',
        type: 'hypothesis',
        content: hypMatch[1].trim(),
        confidence: extractHighestConfidence(hypMatch[1]),
        timestamp: new Date(),
      });
    }

    const conclusionMatch = markdownResponse.match(SECTION_PATTERNS.conclusion);
    if (conclusionMatch) {
      const content = conclusionMatch[1].trim();
      const confMatch = content.match(/overall confidence[:\s]+(\d+)%/i);
      overallConfidence = confMatch ? parseInt(confMatch[1], 10) / 100 : 0.7;

      reasoning.push({
        id: 'conclusion-1',
        type: 'conclusion',
        content,
        confidence: overallConfidence,
        timestamp: new Date(),
      });
    }

    const verifyMatch = markdownResponse.match(SECTION_PATTERNS.verification);
    if (verifyMatch) {
      reasoning.push({
        id: 'verify-1',
        type: 'verification',
        content: verifyMatch[1].trim(),
        confidence: 1.0,
        timestamp: new Date(),
      });
    }

    const sources: SourceReference[] = [];
    if (obsMatch) {
      const sourceNames = extractMetricNames(obsMatch[1]);
      sourceNames.forEach((name, idx) => {
        sources.push({
          type: 'metric',
          id: `metric-${idx}`,
          name,
          relevance: 0.9,
        });
      });
    }

    if (reasoning.length === 0) {
      return null;
    }

    return {
      answer: extractMainAnswer(markdownResponse),
      reasoning,
      sources,
      confidence: overallConfidence,
    };
  } catch (error) {
    console.error('Failed to parse reasoning response:', error);
    return null;
  }
}

function extractSources(text: string): string[] {
  const matches = text.matchAll(METRIC_PATTERN);
  return Array.from(matches, m => m[1]);
}

function extractHighestConfidence(text: string): number {
  const matches = Array.from(text.matchAll(CONFIDENCE_PATTERN));
  if (matches.length === 0) {
    return 0.7;
  }

  const confidences = matches.map(m => {
    const num = m[0].match(/\d+/)?.[0];
    return num ? parseInt(num, 10) / 100 : 0.7;
  });

  return Math.max(...confidences);
}

function extractMetricNames(text: string): string[] {
  const matches = text.matchAll(METRIC_PATTERN);
  return Array.from(new Set(Array.from(matches, m => m[1])));
}

function extractMainAnswer(markdown: string): string {
  const conclusionMatch = markdown.match(SECTION_PATTERNS.conclusion);
  if (!conclusionMatch) {
    return markdown;
  }

  const lines = conclusionMatch[1].split('\n').filter(l => l.trim());
  const answerLine = lines.find(l => l.includes('Answer:'));

  if (answerLine) {
    return answerLine.replace(/\*\*Answer:\s*\*\*\s*/i, '').trim();
  }

  return lines[0] || markdown;
}

export function buildReasoningPrompt(basePrompt: string, mode: 'standard' | 'thinking'): string {
  if (mode !== 'thinking') {
    return basePrompt;
  }

  return basePrompt + `

## REASONING STRUCTURE

Please structure your response following this clear reasoning pattern:

## 🔍 Observation
State what data/metrics/context you have available.
Rate your confidence in the data quality (0-100%).

## 📊 Analysis
Analyze the situation:
- What patterns do you see?
- What stands out as unusual?
- What are the key metrics?

## 💡 Hypothesis
List possible explanations, ranked by likelihood:
1. Most likely explanation (confidence: XX%)
2. Alternative explanation (confidence: XX%)
3. Less likely but possible (confidence: XX%)

## ✅ Conclusion
State your final answer with overall confidence level.
Format: **Answer: [Your answer here]**
**Overall Confidence: XX%**

## 🔬 Verification
How can this be verified? What additional data would help?

---

IMPORTANT: Use the emoji headings above exactly as shown to structure your response.
`;
}
