---
name: generate_query
description: Convert English requests into valid PromQL, LogQL, or TraceQL queries
triggers:
  high_signal:
    - "create a query"
    - "write a query"
    - "generate a query"
    - "query for"
    - "give me a query"
    - "show me a query"
  language_specific:
    - "promql"
    - "logql"
    - "traceql"
  function_keywords:
    - "rate"
    - "increase"
    - "sum"
    - "avg"
    - "count"
    - "histogram"
    - "quantile"
    - "topk"
    - "bottomk"
  action_verbs:
    - "how do i query"
    - "how to query"
    - "how can i get"
    - "show me how to"
    - "need a query"
  general:
    - "metrics"
    - "logs"
    - "traces"
    - "errors"
    - "latency"
    - "throughput"
min_confidence: 40
max_steps: 3
score_weights:
  high_signal: 80
  language_specific: 70
  function_keywords: 15
  action_verbs: 40
  general: 10
supports_template: true
template_placeholder: "%s"
---

Your task is to convert English requests into valid %s queries.

Guidelines:
- Generate syntactically correct %s
- Use best practices (avoid high cardinality, use appropriate functions)
- Explain what the query does
- Warn about potential performance issues
- Suggest time range if not specified

Format your response as:
```%s
<query>
```

**Explanation**: <what the query does>
**Performance**: <any warnings about cardinality, time range, etc>
**Usage**: <suggested time range and resolution>
