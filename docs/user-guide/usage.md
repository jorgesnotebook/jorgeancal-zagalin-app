# Using Zagalin

This guide shows you how to use Zagalin's features to enhance your Grafana experience with AI-powered assistance.

## Overview

Zagalin provides multiple ways to interact with AI:

1. **Floating Chat Button** - Quick access from any dashboard
2. **Full Chat Page** - Dedicated interface for longer sessions
3. **Ask Panel** - Inline queries within dashboards

## Floating Chat Interface

### Accessing Floating Chat

The floating chat button appears automatically when viewing dashboards.

**Location**: Bottom-right corner of the screen

**When it appears**:
- ✅ On any dashboard page
- ❌ Not on home page, login, or admin pages

### Opening Chat

1. Click the **orange chat button** (💬)
2. Chat panel slides up from the bottom
3. Ready to receive your questions

### Chat Panel Features

#### Context Badge

Shows current dashboard context:

- **Green badge** 📊: Dashboard context active
- **Orange badge** ⚠️: No dashboard context
- **Blue badge** ℹ️: Message count

**Hover** over badge to see:
- Dashboard name
- Active panel (if any)
- Time range
- Template variables

#### Message Types

**User Messages** (right-aligned, gray background):
```
You: Show me CPU usage over the last hour
```

**Assistant Messages** (left-aligned, transparent):
```
Zagalin: Here's a PromQL query for CPU usage:
rate(cpu_usage[1h])
```

**System Messages** (centered, informational):
```
⚠️ LLM service is not available
```

#### Sending Messages

**Method 1**: Type and click Send button

**Method 2**: Type and press `Cmd+Enter` (Mac) or `Ctrl+Enter` (Windows/Linux)

**Input Features**:
- Auto-expanding text area
- Max height: 200px
- Scroll for longer messages

### Conversation History

#### Viewing Conversations

1. Click **sidebar toggle** (≪) to show conversation list
2. See all past conversations with:
   - Title
   - Last update time
   - Message count

#### Managing Conversations

**Create New Chat**:
- Click "New Chat" button
- Or select "New Chat" from chat panel header

**Switch Conversations**:
- Click any conversation in the sidebar
- Messages load instantly

**Search Conversations**:
- Type in search box (appears when > 5 conversations)
- Filters by title in real-time

**Pin Important Conversations**:
- Click star icon (⭐)
- Pinned conversations stay at top
- Protected from auto-deletion

**Rename Conversation**:
- Click edit icon (✏️)
- Type new title
- Press Enter to save, Escape to cancel

**Delete Conversation**:
- Click trash icon (🗑️)
- Confirm deletion
- Cannot be undone!

### Context-Aware Features

Zagalin automatically understands:

**Dashboard Context**:
```
You: What does this dashboard show?

Zagalin: This is the "Node Exporter Full" dashboard.
It monitors Linux server metrics including:
- CPU usage (top left panel)
- Memory utilization (top right)
- Disk I/O (bottom panels)
...
```

**Panel Context**:
```
You: Explain this panel

Zagalin: This panel shows "CPU Usage" using the
node_cpu_seconds_total metric. The query calculates
the rate of CPU time spent in different modes...
```

**Time Range Context**:
```
You: Show me errors in the last hour

Zagalin: Searching logs from 14:30 to 15:30...
```

## Full Chat Page

### Accessing Full Chat

**Navigation**: Apps → Zagalin → Chat

**When to Use**:
- Longer conversations
- Multiple related questions
- When you need more screen space

### Features

**Same as floating chat**, plus:
- Full-screen interface
- No panel size limitations
- Better for complex discussions
- Conversation history in sidebar

### Welcome Screen

When no messages exist:
- Zagalin logo
- Welcome message
- Suggested prompts (future feature)

## Common Use Cases

### 1. Query Generation

**Natural Language → PromQL/LogQL**

```
You: Show me memory usage above 80%

Zagalin: Here's the PromQL query:

(node_memory_MemTotal_bytes - node_memory_MemAvailable_bytes)
/ node_memory_MemTotal_bytes * 100 > 80

[Copy] [Explore]
```

**Actions Available**:
- **Copy**: Copy query to clipboard
- **Explore**: Open in Grafana Explore

### 2. Dashboard Analysis

**Understanding Dashboards**

```
You: What is this dashboard for?

Zagalin: This is a Kubernetes cluster monitoring dashboard.
It tracks:
- Pod health and status
- Node resource utilization
- Network traffic
- Storage capacity

Key panels:
1. "Running Pods" - Shows active pods count
2. "CPU by Container" - Resource usage breakdown
3. "Network I/O" - Bandwidth consumption
```

### 3. Panel Explanation

**Understanding Metrics**

```
You: What does this panel measure?

Zagalin: This panel displays "Request Rate" using:

sum(rate(http_requests_total[5m])) by (status)

It measures:
- Total HTTP requests per second
- Grouped by response status code
- Averaged over 5-minute windows

Status codes:
- 2xx: Successful requests
- 4xx: Client errors
- 5xx: Server errors
```

### 4. Troubleshooting

**Guided Problem Solving**

```
You: My application is slow

Zagalin: Let's investigate. I'll check several metrics:

1. Response Time
   Query: histogram_quantile(0.95, http_request_duration_seconds_bucket)
   Current: 2.3s (elevated)

2. Error Rate
   Query: rate(http_requests_total{status=~"5.."}[5m])
   Current: 0.02 (normal)

3. Database Connections
   Query: mysql_global_status_threads_connected
   Current: 487/500 (near limit!)

Diagnosis: High response times likely due to database
connection pool exhaustion.

Recommended Actions:
- Increase connection pool size
- Add connection pooling timeout
- Review slow queries
```

### 5. Query Optimization

**Improving Existing Queries**

```
You: Can you optimize this query?
rate(http_requests_total[5m])

Zagalin: Here's an optimized version:

# Original (slow)
rate(http_requests_total[5m])

# Optimized (faster)
sum(rate(http_requests_total[5m])) by (instance)

Improvements:
1. Added sum() to aggregate data
2. Group by instance for clarity
3. Reduced data points returned

Performance impact: ~60% faster query execution
```

## Keyboard Shortcuts

| Shortcut | Action |
|----------|--------|
| `Cmd/Ctrl + Enter` | Send message |
| `Escape` | Close floating chat |
| `Tab` | Navigate between UI elements |
| `Enter` | Save when renaming conversation |
| `Escape` | Cancel when renaming conversation |

## Chat Etiquette

### Effective Prompts

**Good Prompts** ✅:
- "Show me CPU usage over the last hour"
- "Explain the 'API Response Time' panel"
- "Why is memory usage increasing?"
- "Generate a query for disk I/O by mount point"

**Unclear Prompts** ❌:
- "Fix this" (no context)
- "Help" (too vague)
- "Query" (incomplete request)

### Getting Better Answers

**Be Specific**:
```
❌ "Show me metrics"
✅ "Show me CPU metrics for the last 24 hours"
```

**Provide Context**:
```
❌ "This is broken"
✅ "The 'API Latency' panel shows no data since 2pm"
```

**Ask Follow-ups**:
```
You: Show me error rate
Zagalin: [provides query]
You: Can you break that down by service?
Zagalin: [provides enhanced query]
```

## Customization

### Personality Presets

Configure in: Apps → Zagalin → Configuration

**Available Presets**:

1. **Helpful** (Default)
   - Balanced and practical
   - Good for most users

2. **Technical**
   - Detailed explanations
   - Assumes SRE/DevOps knowledge

3. **Beginner-Friendly**
   - Patient explanations
   - Defines technical terms

4. **Concise**
   - Brief responses
   - Gets straight to the point

5. **Custom**
   - Write your own instructions
   - Example: "Always include links to documentation"

### Skills Configuration

Enable/disable specific capabilities:

- ✅ **Explain Panel** - Panel analysis
- ✅ **Generate Queries** - PromQL/LogQL generation
- ✅ **Troubleshooting** - Problem diagnosis
- ⚠️ **Vector Search** - Semantic search (requires setup)
- ✅ **Function Calling** - Structured tool execution

### LLM Parameters

**Temperature** (0.0 - 1.0):
- 0.0: Deterministic, factual
- 0.5: Balanced (default)
- 1.0: Creative, varied

**Max Tokens** (1000 - 4000):
- 1000: Brief responses
- 2000: Standard (default)
- 4000: Detailed explanations

## Tips and Tricks

### 1. Use Context to Your Advantage

Navigate to relevant dashboard first:
```
# On Kubernetes dashboard
You: How many pods are running?
# Zagalin has full dashboard context
```

### 2. Ask for Multiple Formats

```
You: Show me CPU usage

Zagalin: [provides PromQL]

You: Can you also show LogQL for CPU logs?

Zagalin: [provides LogQL variant]
```

### 3. Refine Queries Iteratively

```
You: Show me errors
Zagalin: [basic query]

You: Only for the last hour
Zagalin: [adds time range]

You: Group by service
Zagalin: [adds grouping]
```

### 4. Save Important Conversations

Pin conversations you reference frequently:
- Click star icon ⭐
- Easy to find later
- Won't be auto-deleted

### 5. Use Conversation History

Resume previous discussions:
- Click past conversation
- Continue where you left off
- Full context preserved

## Troubleshooting Usage Issues

### Chat Button Not Appearing

**Causes**:
- Not on a dashboard page
- Plugin not enabled
- JavaScript error

**Solutions**:
1. Navigate to a dashboard
2. Check: Apps → Zagalin → Enable
3. Check browser console for errors

### No LLM Responses

**Causes**:
- Grafana LLM App not configured
- API key missing/invalid
- Network issues

**Solutions**:
1. Configure: Plugins → LLM App
2. Add API key
3. Test connection

### Messages Not Saving

**Causes**:
- localStorage disabled
- Browser in incognito mode
- Storage quota exceeded

**Solutions**:
1. Enable localStorage in browser settings
2. Use regular browser window
3. Delete old conversations

### Slow Responses

**Causes**:
- Large dashboard context
- High API latency
- Rate limiting

**Solutions**:
1. Reduce context size (future feature)
2. Check network connection
3. Wait for rate limit reset

## Privacy and Data

### What Gets Stored

**Locally** (localStorage):
- Conversation history
- Chat panel position/size
- Plugin configuration

**Sent to LLM**:
- Your messages
- Dashboard/panel context (if enabled)
- System prompts

**Never Stored**:
- Credentials or API keys
- Sensitive data (unless you include it in messages)

### Data Retention

- **Conversations**: Until you delete them
- **Auto-prune**: Max 50 conversations
- **Message limit**: Max 100 per conversation

### Clearing Data

**Clear Conversations**:
- Delete individual conversations
- Or clear browser localStorage

**Clear Configuration**:
- Reset in Configuration page
- Or clear browser localStorage

## Getting Help

### In-App Help

- Hover over badges for tooltips
- Check context badge for current state
- Review error messages

### External Resources

- [Troubleshooting Guide](./troubleshooting.md)
- [GitHub Issues](https://github.com/jorgesnotebook/jorgeancal-zagalin-app/issues)
- [Community Forum](https://community.grafana.com/)
- [Documentation](../README.md)

## Next Steps

- [Configuration Guide](./configuration.md)
- [Troubleshooting](./troubleshooting.md)
- [Advanced Features](./advanced-features.md)
- [FAQ](./faq.md)

## Feedback

Help improve Zagalin:
- [Report bugs](https://github.com/jorgesnotebook/jorgeancal-zagalin-app/issues)
- [Request features](https://github.com/jorgesnotebook/jorgeancal-zagalin-app/issues)
- [Share feedback](https://community.grafana.com/)
