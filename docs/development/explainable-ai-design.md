# Explainable AI Response System Design

## Overview

This document outlines the design for implementing explainable, structured AI responses with clear reasoning patterns in Zagalin.

## Current State

**What we have:**

- Basic "thinking" mode that adjusts temperature/tokens
- Enhanced system prompt for deeper reasoning
- Streaming responses with markdown rendering
- Artifact display system (evidence cards)
- Tool calling with structured function execution

**What's missing:**

- Structured reasoning steps visible to users
- Confidence scores for answers
- Source attribution (which data sources were used)
- Step-by-step explanation of decision process
- Visual reasoning pattern display

## Goals

1. **Transparency**: Users understand HOW the AI arrived at conclusions
2. **Trust**: Users can verify reasoning with confidence scores and sources
3. **Debuggability**: Users can identify where reasoning might be flawed
4. **Education**: Users learn best practices by observing AI reasoning

## Design

### 1. Structured Response Format

Add new response types to support structured reasoning:

```typescript
// src/types/explainableAI.ts

export interface ReasoningStep {
  id: string;
  type: 'observation' | 'hypothesis' | 'analysis' | 'conclusion';
  content: string;
  confidence: number; // 0-1
  timestamp: Date;
  sources?: string[]; // References to context used
}

export interface SourceReference {
  type: 'dashboard' | 'panel' | 'metric' | 'log' | 'trace' | 'documentation';
  id: string;
  name: string;
  relevance: number; // 0-1
}

export interface ExplainableResponse {
  answer: string; // Final answer text
  reasoning: ReasoningStep[]; // Step-by-step reasoning
  sources: SourceReference[]; // Data sources used
  confidence: number; // Overall confidence 0-1
  alternativeApproaches?: string[]; // Other ways to solve this
  caveats?: string[]; // Limitations or assumptions
}

export interface StreamChunkWithReasoning extends StreamChunk {
  reasoning_step?: ReasoningStep;
  source_ref?: SourceReference;
  confidence?: number;
}
```

### 2. Backend Changes

#### A. Enhanced LLM Prompting Strategy

**Option 1: Chain-of-Thought Prompting** (Immediate, no schema changes)

```go
// pkg/plugin/assistant_prompts.go

func BuildReasoningSystemPrompt(skill string) string {
    base := BuildSystemPrompt(skill, context)

    reasoning := `

## Reasoning Structure

When responding, structure your thinking as follows:

**1. Observation**
- What data/metrics/logs are available?
- What is the current state?

**2. Analysis**
- What patterns or anomalies do you see?
- What might be causing the issue?

**3. Hypothesis**
- What are the most likely explanations?
- Rank by probability

**4. Conclusion**
- What's the recommended action?
- What's the confidence level?

**5. Verification**
- How can this be tested?
- What additional data would help?

Format your response with clear sections using markdown headers.
`

    return base + reasoning
}
```

**Option 2: Structured Output with JSON** (Better, requires schema)

```go
// Use Claude/GPT's structured output feature
func BuildStructuredReasoningRequest() string {
    return `
You must respond with valid JSON matching this schema:
{
  "answer": "The main answer to the question",
  "reasoning_steps": [
    {
      "type": "observation",
      "content": "What I observed...",
      "confidence": 0.9,
      "sources": ["dashboard_uid", "metric_name"]
    }
  ],
  "overall_confidence": 0.85,
  "sources_used": [
    {
      "type": "metric",
      "name": "http_requests_total",
      "relevance": 0.9
    }
  ],
  "caveats": ["Assumption: traffic is evenly distributed"]
}
`
}
```

#### B. Confidence Scoring

Add confidence estimation to responses:

```go
// pkg/plugin/confidence.go

type ConfidenceEstimator struct {
    contextQuality float64    // How much relevant context is available
    skillMatch     float64    // How well does skill match the question
    dataFreshness  float64    // How recent is the data
}

func (ce *ConfidenceEstimator) Calculate() float64 {
    // Weighted average
    return (ce.contextQuality * 0.4) +
           (ce.skillMatch * 0.3) +
           (ce.dataFreshness * 0.3)
}

func EstimateConfidence(ctx AssistantContext, skill string) float64 {
    estimator := &ConfidenceEstimator{}

    // Calculate context quality
    if ctx.Dashboard != nil && len(ctx.Dashboard.Panels) > 0 {
        estimator.contextQuality = 0.8
    } else {
        estimator.contextQuality = 0.3
    }

    // Calculate skill match
    if skill != "" {
        estimator.skillMatch = 0.9
    } else {
        estimator.skillMatch = 0.6
    }

    // Calculate data freshness
    if ctx.TimeRange != nil {
        // Check if time range is recent
        estimator.dataFreshness = 0.7
    } else {
        estimator.dataFreshness = 0.5
    }

    return estimator.Calculate()
}
```

#### C. Source Tracking

Track which data sources are referenced:

```go
// pkg/plugin/source_tracker.go

type SourceTracker struct {
    sources []SourceReference
}

func (st *SourceTracker) TrackDashboard(ctx AssistantContext) {
    if ctx.Dashboard != nil {
        st.sources = append(st.sources, SourceReference{
            Type:      "dashboard",
            ID:        ctx.Dashboard.UID,
            Name:      ctx.Dashboard.Title,
            Relevance: 1.0,
        })
    }
}

func (st *SourceTracker) TrackPanels(ctx AssistantContext) {
    if ctx.Dashboard == nil {
        return
    }

    for _, panel := range ctx.Dashboard.Panels {
        for _, target := range panel.Targets {
            if target.Expr != "" {
                st.sources = append(st.sources, SourceReference{
                    Type:      "metric",
                    ID:        target.RefID,
                    Name:      target.Expr,
                    Relevance: 0.8,
                })
            }
        }
    }
}

func (st *SourceTracker) GetSources() []SourceReference {
    return st.sources
}
```

### 3. Frontend Changes

#### A. Reasoning Step Display Component

```typescript
// src/components/FloatingChat/ReasoningSteps.tsx

interface ReasoningStepsProps {
  steps: ReasoningStep[];
  collapsed?: boolean;
}

export function ReasoningSteps({ steps, collapsed = true }: ReasoningStepsProps) {
  const [isExpanded, setIsExpanded] = useState(!collapsed);

  const getStepIcon = (type: string) => {
    switch (type) {
      case 'observation':
        return '';
      case 'hypothesis':
        return '';
      case 'analysis':
        return '';
      case 'conclusion':
        return '';
      default:
        return '';
    }
  };

  const getConfidenceColor = (confidence: number) => {
    if (confidence >= 0.8) return 'green';
    if (confidence >= 0.6) return 'yellow';
    return 'orange';
  };

  return (
    <div className={s.reasoningSteps}>
      <div className={s.reasoningHeader} onClick={() => setIsExpanded(!isExpanded)}>
        <Icon name={isExpanded ? 'angle-down' : 'angle-right'} />
        <span>Reasoning Process ({steps.length} steps)</span>
      </div>

      {isExpanded && (
        <div className={s.stepsContainer}>
          {steps.map((step, idx) => (
            <div key={step.id} className={s.reasoningStep}>
              <div className={s.stepHeader}>
                <span className={s.stepIcon}>{getStepIcon(step.type)}</span>
                <span className={s.stepType}>{step.type}</span>
                <Badge text={`${Math.round(step.confidence * 100)}%`} color={getConfidenceColor(step.confidence)} />
              </div>
              <div className={s.stepContent}>{step.content}</div>
              {step.sources && step.sources.length > 0 && (
                <div className={s.stepSources}>
                  <Icon name="link" size="sm" />
                  {step.sources.join(', ')}
                </div>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
```

#### B. Confidence Indicator

```typescript
// src/components/FloatingChat/ConfidenceIndicator.tsx

interface ConfidenceIndicatorProps {
  confidence: number; // 0-1
  showLabel?: boolean;
}

export function ConfidenceIndicator({ confidence, showLabel = true }: ConfidenceIndicatorProps) {
  const percentage = Math.round(confidence * 100);

  const getColor = () => {
    if (confidence >= 0.8) return 'green';
    if (confidence >= 0.6) return 'yellow';
    if (confidence >= 0.4) return 'orange';
    return 'red';
  };

  const getLabel = () => {
    if (confidence >= 0.8) return 'High confidence';
    if (confidence >= 0.6) return 'Medium confidence';
    if (confidence >= 0.4) return 'Low confidence';
    return 'Very low confidence';
  };

  return (
    <div className={s.confidenceIndicator}>
      <div className={s.confidenceBar}>
        <div
          className={s.confidenceFill}
          style={{
            width: `${percentage}%`,
            backgroundColor: `var(--${getColor()})`,
          }}
        />
      </div>
      {showLabel && (
        <span className={s.confidenceLabel}>
          {getLabel()} ({percentage}%)
        </span>
      )}
    </div>
  );
}
```

#### C. Source Attribution Panel

```typescript
// src/components/FloatingChat/SourceAttribution.tsx

interface SourceAttributionProps {
  sources: SourceReference[];
}

export function SourceAttribution({ sources }: SourceAttributionProps) {
  const getSourceIcon = (type: string) => {
    switch (type) {
      case 'dashboard':
        return 'apps';
      case 'panel':
        return 'panel-add';
      case 'metric':
        return 'graph-bar';
      case 'log':
        return 'file-alt';
      case 'trace':
        return 'process';
      default:
        return 'link';
    }
  };

  return (
    <div className={s.sourceAttribution}>
      <h4>Sources Used</h4>
      <div className={s.sourceList}>
        {sources.map((source, idx) => (
          <div key={idx} className={s.sourceItem}>
            <Icon name={getSourceIcon(source.type)} />
            <span className={s.sourceName}>{source.name}</span>
            <Badge text={`${Math.round(source.relevance * 100)}%`} color="blue" />
          </div>
        ))}
      </div>
    </div>
  );
}
```

#### D. Enhanced Message Display

Update ChatPanel to show reasoning components:

```typescript
// src/components/FloatingChat/ChatPanel.tsx (update)

{
  messages.map((message, idx) => (
    <div key={idx} className={`${s.message} ${s.assistantMessage}`}>
      {/* Main answer */}
      <div className={s.messageContent} dangerouslySetInnerHTML={{ __html: sanitizeMarkdown(message.content) }} />

      {/* Confidence indicator (if available) */}
      {message.confidence !== undefined && <ConfidenceIndicator confidence={message.confidence} />}

      {/* Reasoning steps (collapsible) */}
      {message.reasoning && message.reasoning.length > 0 && (
        <ReasoningSteps steps={message.reasoning} collapsed={true} />
      )}

      {/* Source attribution */}
      {message.sources && message.sources.length > 0 && <SourceAttribution sources={message.sources} />}

      {/* Caveats/limitations */}
      {message.caveats && message.caveats.length > 0 && (
        <Alert severity="info" title="Important Considerations">
          <ul>
            {message.caveats.map((caveat, i) => (
              <li key={i}>{caveat}</li>
            ))}
          </ul>
        </Alert>
      )}
    </div>
  ));
}
```

## Implementation Phases

### Phase 1: Foundation (Week 1)

- Add new TypeScript types for explainable responses
- Implement chain-of-thought prompting in backend
- Add confidence estimator
- Create basic ReasoningSteps component

### Phase 2: Source Tracking (Week 2)

- Implement source tracker in backend
- Add source attribution to LLM requests
- Create SourceAttribution component
- Update message storage to include sources

### Phase 3: Structured Output (Week 3)

- Implement JSON schema-based structured output
- Parse reasoning steps from LLM response
- Handle streaming of structured data
- Add confidence calculation to responses

### Phase 4: UI Polish (Week 4)

- Design and implement visual reasoning display
- Add collapsible reasoning sections
- Implement confidence indicators
- Add user preference for showing/hiding reasoning

### Phase 5: Advanced Features (Future)

- Alternative approach suggestions
- Interactive reasoning (users can query specific steps)
- Reasoning export for documentation
- A/B testing different reasoning strategies

## Example Output

```
User: "Why is my API latency spiking?"

Assistant Response:

[Main Answer]
Your API latency is spiking due to increased database query time.
The primary cause is missing indexes on the users table.

[Confidence: 85% - High confidence ]

[Reasoning Process] (collapsed by default)
   Observation (95% confidence)
     - API p95 latency increased from 200ms to 1.2s
     - Database query time increased 6x
     - No changes in traffic volume
     Sources: http_request_duration_seconds, db_query_duration_seconds

   Analysis (90% confidence)
     - 80% of slow queries target users table
     - Query patterns show full table scans
     - No recent schema changes detected
     Sources: slow_query_log, users_table_metrics

   Hypothesis (80% confidence)
     1. Missing indexes (90% likely)
     2. Lock contention (30% likely)
     3. Connection pool exhaustion (20% likely)

   Conclusion (85% confidence)
     Missing indexes are the primary cause based on:
     - Query execution plans show sequential scans
     - High correlation with query patterns
     - No evidence of locking or connection issues

[Sources Used]
   http_request_duration_seconds (Relevance: 95%)
   db_query_duration_seconds (Relevance: 90%)
   slow_query_log (Relevance: 85%)

[ Important Considerations]
  • Analysis based on last 1 hour of data
  • Assumes database metrics are accurate
  • Recommendation needs testing in staging first
```

## Benefits

1. **Transparency**: Users see the full decision process
2. **Trust**: Confidence scores help users assess reliability
3. **Debuggability**: Can identify flawed reasoning steps
4. **Education**: Users learn troubleshooting methodology
5. **Compliance**: Audit trail for AI decisions

## Technical Considerations

### Performance

- Structured output adds ~10-20% to response time
- Implement caching for confidence calculations
- Use streaming to show reasoning as it's generated

### Token Usage

- Structured prompts use 200-300 more tokens
- Monitor and adjust based on cost/benefit
- Consider making it optional per-request

### Model Compatibility

- Claude: Excellent at chain-of-thought
- GPT-4: Good with structured output
- Smaller models: May need simplified reasoning

### Storage

- Reasoning steps increase message size by ~2-3x
- Consider separate storage for reasoning data
- Implement TTL for old reasoning data

## References

- [Chain-of-Thought Prompting](https://arxiv.org/abs/2201.11903)
- [Structured Output with LLMs](https://platform.openai.com/docs/guides/structured-outputs)
- [Explainable AI Guidelines](https://arxiv.org/abs/2011.07876)
- [Claude Extended Thinking](https://docs.anthropic.com/en/docs/build-with-claude/extended-thinking)
