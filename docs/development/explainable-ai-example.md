# Explainable AI Implementation Example

## Quick Start: Adding Reasoning to Responses

### Step 1: Simple Chain-of-Thought (Immediate Implementation)

The easiest way to add explainable reasoning is through **enhanced prompting**. No schema changes needed!

#### Backend Prompt Enhancement

```go
// pkg/plugin/assistant_prompts.go

func BuildReasoningPrompt(skill string, message string, ctx AssistantContext) string {
    prompt := fmt.Sprintf(`
User Question: %s

Please structure your response following this reasoning pattern:

##  Observation
First, state what data/metrics/context you have available.
Rate your confidence in the data quality (0-100%%).

##  Analysis
Analyze the situation:
- What patterns do you see?
- What stands out as unusual?
- What are the key metrics?

##  Hypothesis
List possible explanations, ranked by likelihood:
1. Most likely explanation (confidence: XX%%)
2. Alternative explanation (confidence: XX%%)
3. Less likely but possible (confidence: XX%%)

##  Conclusion
State your final answer with overall confidence level.

##  Verification
How can this be verified? What additional data would help?

---

Respond in markdown format. Use the emoji headings above to structure your answer.
`, message)

    return prompt
}
```

#### Example Response (Auto-formatted)

When you use this prompt, Claude naturally produces:

```markdown
## Observation

I can see from your dashboard context:

- API latency (p95) increased from 200ms to 1.2s over the last hour
- Database query duration increased 6x in the same period
- No change in request volume (steady at 1000 req/min)
- Using Prometheus metrics: `http_request_duration_seconds`, `db_query_duration_seconds`

**Data Quality Confidence: 90%** - Recent data from reliable sources

## Analysis

Key findings:

- **80% of slow queries** target the users table
- Query plans show **full table scans** instead of index usage
- No evidence of lock contention (lock wait time is normal)
- Connection pool utilization is at 60% (healthy)

## Hypothesis

Ranked by likelihood:

1. **Missing database indexes (Confidence: 85%)**

   - Evidence: Full table scans in query plans
   - Timing: Correlates with increased data volume
   - Impact: 6x slowdown matches expected behavior

2. **Query N+1 problem (Confidence: 40%)**

   - Would explain multiplier effect
   - But connection pool shows no saturation
   - Less likely given metrics

3. **Network latency (Confidence: 10%)**
   - Would affect all queries equally
   - Only users table affected
   - Very unlikely

## Conclusion

**Answer: Add indexes to the users table**

**Overall Confidence: 80%**

The primary cause is missing indexes on frequently queried columns.
This is a high-confidence recommendation based on query execution
plans and performance correlation.

## Verification

To verify:

1. Check query execution plans: `EXPLAIN SELECT * FROM users WHERE email = ?`
2. Review recent schema changes in the last 7 days
3. Test adding indexes in staging environment first

Additional helpful data:

- Slow query log from database
- Recent deployment history
- Table growth rate over time
```

### Step 2: Parsing Structured Responses (Frontend)

Create a simple parser to extract reasoning sections:

```typescript
// src/services/reasoningParser.ts

import { ReasoningStep, SourceReference, ExplainableResponse } from '../types/explainableAI';

const SECTION_PATTERNS = {
  observation: /##\s*\s*Observation([\s\S]*?)(?=##|$)/i,
  analysis: /##\s*\s*Analysis([\s\S]*?)(?=##|$)/i,
  hypothesis: /##\s*\s*Hypothesis([\s\S]*?)(?=##|$)/i,
  conclusion: /##\s*\s*Conclusion([\s\S]*?)(?=##|$)/i,
  verification: /##\s*\s*Verification([\s\S]*?)(?=##|$)/i,
};

const CONFIDENCE_PATTERN = /confidence[:\s]+(\d+)%/gi;

export function parseReasoningResponse(markdownResponse: string): ExplainableResponse | null {
  try {
    const reasoning: ReasoningStep[] = [];
    let overallConfidence = 0.7; // Default

    // Extract observation
    const obsMatch = markdownResponse.match(SECTION_PATTERNS.observation);
    if (obsMatch) {
      const content = obsMatch[1].trim();
      const confMatch = content.match(CONFIDENCE_PATTERN);
      const confidence = confMatch ? parseInt(confMatch[0].match(/\d+/)?.[0] || '70') / 100 : 0.9;

      reasoning.push({
        id: 'obs-1',
        type: 'observation',
        content,
        confidence,
        timestamp: new Date(),
        sources: extractSources(content),
      });
    }

    // Extract analysis
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

    // Extract hypothesis
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

    // Extract conclusion
    const conclusionMatch = markdownResponse.match(SECTION_PATTERNS.conclusion);
    if (conclusionMatch) {
      const content = conclusionMatch[1].trim();
      const confMatch = content.match(/overall confidence[:\s]+(\d+)%/i);
      overallConfidence = confMatch ? parseInt(confMatch[1]) / 100 : 0.7;

      reasoning.push({
        id: 'conclusion-1',
        type: 'conclusion',
        content,
        confidence: overallConfidence,
        timestamp: new Date(),
      });
    }

    // Extract verification
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

    // Extract sources from observation section
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
  const metricPattern = /`([a-z_][a-z0-9_]*)`/gi;
  const matches = text.matchAll(metricPattern);
  return Array.from(matches, (m) => m[1]);
}

function extractHighestConfidence(text: string): number {
  const matches = Array.from(text.matchAll(CONFIDENCE_PATTERN));
  if (matches.length === 0) return 0.7;

  const confidences = matches.map((m) => {
    const num = m[0].match(/\d+/)?.[0];
    return num ? parseInt(num) / 100 : 0.7;
  });

  return Math.max(...confidences);
}

function extractMetricNames(text: string): string[] {
  const metricPattern = /`([a-z_][a-z0-9_]*)`/gi;
  const matches = text.matchAll(metricPattern);
  return Array.from(new Set(Array.from(matches, (m) => m[1])));
}

function extractMainAnswer(markdown: string): string {
  const conclusionMatch = markdown.match(SECTION_PATTERNS.conclusion);
  if (!conclusionMatch) return markdown;

  const lines = conclusionMatch[1].split('\n').filter((l) => l.trim());
  const answerLine = lines.find((l) => l.includes('Answer:'));

  if (answerLine) {
    return answerLine.replace(/\*\*Answer:\s*\*\*\s*/i, '').trim();
  }

  return lines[0] || markdown;
}
```

### Step 3: Update Message Storage

```typescript
// src/services/conversationStorage.ts (extend StoredMessage)

export interface StoredMessage {
  id: string;
  role: 'user' | 'assistant' | 'system';
  content: string;
  timestamp: Date;
  tokens?: number;
  cost?: number;
  artifacts?: any[];

  // NEW: Explainable AI fields
  reasoning?: ReasoningStep[];
  sources?: SourceReference[];
  confidence?: number;
  caveats?: string[];
}
```

### Step 4: Update Chat Display

```typescript
// src/components/FloatingChat/ChatPanel.tsx (in handleSend or streaming callback)

// After receiving complete response
const explainableResponse = parseReasoningResponse(displayedContent);

if (explainableResponse) {
  const assistantMessage: ConversationMessage = {
    id: generateId(),
    role: 'assistant',
    content: explainableResponse.answer,
    timestamp: new Date(),
    reasoning: explainableResponse.reasoning,
    sources: explainableResponse.sources,
    confidence: explainableResponse.confidence,
  };

  await addMessage(assistantMessage);
} else {
  // Fallback: save as regular message
  const assistantMessage: ConversationMessage = {
    id: generateId(),
    role: 'assistant',
    content: displayedContent,
    timestamp: new Date(),
  };

  await addMessage(assistantMessage);
}
```

## Testing the Feature

### 1. Enable Reasoning Mode

Add a toggle to ChatPanel:

```typescript
const [showReasoning, setShowReasoning] = useState(true);

// In settings or UI
<Switch
  label="Show reasoning process"
  value={showReasoning}
  onChange={(e) => setShowReasoning(e.currentTarget.checked)}
/>;
```

### 2. Test Queries

Try these test queries to see reasoning in action:

```
1. "Why is my API latency increasing?"
   → Should show observation, analysis, hypothesis pattern

2. "How do I optimize this PromQL query: rate(http_requests_total[5m])?"
   → Should show analysis of query and optimization steps

3. "What's causing high CPU usage in my service?"
   → Should show systematic troubleshooting reasoning

4. "Explain this error: 'connection pool exhausted'"
   → Should show hypothesis ranking and verification steps
```

### 3. Verify Output

Check that:

- Reasoning sections are properly parsed
- Confidence scores are extracted
- Source metrics are identified
- Main answer is clearly stated
- Verification steps are provided

## Next Steps

1. **Create ReasoningSteps component** (from design doc)
2. **Add confidence indicator UI**
3. **Implement source attribution panel**
4. **Add user preferences** for showing/hiding reasoning
5. **Track metrics** on reasoning quality

## Benefits of This Approach

**Quick to implement** - No backend schema changes
**Works with existing models** - Just prompt engineering
**Backwards compatible** - Falls back gracefully
**User-friendly** - Natural markdown formatting
**Debuggable** - Clear structure makes issues obvious

## Limitations

**Not guaranteed** - LLM might not always follow format
**Parsing fragility** - Regex-based parsing can break
**Token overhead** - Uses more tokens per request
**Model dependent** - Works best with Claude/GPT-4

## Migration Path to Structured JSON

Once this is proven, you can enhance it with:

1. **Structured output schema** - Force JSON format
2. **Streaming structured data** - Send reasoning steps as they're generated
3. **Real-time confidence calculation** - Backend calculates, not extracted
4. **Source tracking** - Backend tracks actual sources used

See `explainable-ai-design.md` for the full roadmap.
