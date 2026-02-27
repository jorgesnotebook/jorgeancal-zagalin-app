package plugin

// Slack Socket Mode integration — outbound WebSocket only, no inbound HTTP.
//
// The bot connects to Slack's WSS endpoint using the App-Level Token (xapp-…)
// so Grafana never needs a public URL. Events (slash commands, app_mentions)
// arrive over the socket; responses are posted back via chat.postMessage using
// the Bot User OAuth Token (xoxb-…).
//
// Required Slack app configuration:
//   - Socket Mode:   enabled (Settings → Socket Mode)
//   - App-Level Token scopes: connections:write
//   - Bot Token scopes:       app_mentions:read, commands, chat:write
//   - Event subscriptions:    app_mention  (under Events API)
//   - Slash command:          /zagalin (or any name you choose)

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

const (
	slackAPIBase          = "https://slack.com/api"
	slackMaxReconnectWait = 5 * time.Minute
	slackToolLoopMaxSteps = 5 // max LLM→tool iterations per Slack message
)

// ── Slack envelope types ──────────────────────────────────────────────────────

type slackEnvelope struct {
	EnvelopeID             string          `json:"envelope_id"`
	Type                   string          `json:"type"`
	Payload                json.RawMessage `json:"payload"`
	AcceptsResponsePayload bool            `json:"accepts_response_payload"`
}

type slackSlashPayload struct {
	Command     string `json:"command"`
	Text        string `json:"text"`
	UserID      string `json:"user_id"`
	UserName    string `json:"user_name"`
	ChannelID   string `json:"channel_id"`
	ResponseURL string `json:"response_url"`
}

type slackEventPayload struct {
	Event struct {
		Type     string `json:"type"`
		Text     string `json:"text"`
		Channel  string `json:"channel"`
		UserID   string `json:"user"`
		TS       string `json:"ts"`
		ThreadTS string `json:"thread_ts"`
	} `json:"event"`
}

// ── SlackBot ──────────────────────────────────────────────────────────────────

// SlackBot manages the Slack Socket Mode WebSocket connection and routes
// incoming events to the AI assistant.
type SlackBot struct {
	settings *Settings
	app      *App
	cancel   context.CancelFunc
	done     chan struct{}
}

func newSlackBot(settings *Settings, app *App) *SlackBot {
	return &SlackBot{
		settings: settings,
		app:      app,
		done:     make(chan struct{}),
	}
}

// Start launches the bot in a background goroutine. Safe to call once.
func (b *SlackBot) Start(parentCtx context.Context) {
	ctx, cancel := context.WithCancel(parentCtx)
	b.cancel = cancel
	go b.run(ctx)
}

// Stop shuts down the bot and waits for the goroutine to exit.
func (b *SlackBot) Stop() {
	if b.cancel != nil {
		b.cancel()
	}
	<-b.done
}

// run is the main loop: connects to Slack and reconnects on failure.
func (b *SlackBot) run(ctx context.Context) {
	defer close(b.done)

	backoff := 2 * time.Second

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		conn, err := b.connect(ctx)
		if err != nil {
			backend.Logger.Warn("Slack Socket Mode: connection failed, retrying",
				"error", err,
				"backoffSeconds", backoff.Seconds(),
			)
			select {
			case <-ctx.Done():
				return
			case <-time.After(backoff):
				if backoff < slackMaxReconnectWait {
					backoff *= 2
				}
			}
			continue
		}

		backend.Logger.Info("Slack Socket Mode: connected")
		backoff = 2 * time.Second // reset after successful connection

		if err := b.handleMessages(ctx, conn); err != nil && ctx.Err() == nil {
			backend.Logger.Warn("Slack Socket Mode: connection lost, reconnecting", "error", err)
		}
		conn.Close()
	}
}

// connect obtains a fresh WSS URL from Slack and dials the WebSocket.
func (b *SlackBot) connect(ctx context.Context) (*websocket.Conn, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		slackAPIBase+"/apps.connections.open", nil)
	if err != nil {
		return nil, fmt.Errorf("build connections.open request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+b.settings.SlackAppToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("apps.connections.open: %w", err)
	}
	defer resp.Body.Close()

	var openResp struct {
		OK    bool   `json:"ok"`
		URL   string `json:"url"`
		Error string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&openResp); err != nil {
		return nil, fmt.Errorf("decode connections.open response: %w", err)
	}
	if !openResp.OK {
		return nil, fmt.Errorf("Slack API error: %s", openResp.Error)
	}

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, openResp.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("websocket dial: %w", err)
	}
	return conn, nil
}

// handleMessages reads messages from the WebSocket until the connection closes
// or the context is cancelled.
func (b *SlackBot) handleMessages(ctx context.Context, conn *websocket.Conn) error {
	// Close the connection when the context is cancelled so ReadMessage unblocks.
	go func() {
		<-ctx.Done()
		conn.Close()
	}()

	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			if ctx.Err() != nil {
				return nil // clean shutdown
			}
			return fmt.Errorf("read message: %w", err)
		}

		var envelope slackEnvelope
		if err := json.Unmarshal(data, &envelope); err != nil {
			backend.Logger.Warn("Slack: failed to parse envelope", "error", err)
			continue
		}

		switch envelope.Type {
		case "hello":
			backend.Logger.Debug("Slack Socket Mode: hello received")
		case "disconnect":
			backend.Logger.Info("Slack Socket Mode: server requested disconnect")
			return fmt.Errorf("server requested disconnect")
		case "slash_commands":
			b.ackEnvelope(conn, envelope.EnvelopeID, "")
			go b.handleSlashCommand(ctx, envelope.Payload)
		case "events_api":
			b.ackEnvelope(conn, envelope.EnvelopeID, "")
			go b.handleEventAPI(ctx, envelope.Payload)
		default:
			backend.Logger.Debug("Slack: unhandled envelope type", "type", envelope.Type)
		}
	}
}

// ackEnvelope sends the mandatory acknowledgement for each Socket Mode envelope.
// Must be called within 3 seconds or Slack will resend the event.
func (b *SlackBot) ackEnvelope(conn *websocket.Conn, envelopeID, immediateText string) {
	ack := map[string]interface{}{"envelope_id": envelopeID}
	if immediateText != "" {
		ack["payload"] = map[string]string{"text": immediateText}
	}
	data, _ := json.Marshal(ack)
	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		backend.Logger.Warn("Slack: failed to send ACK",
			"error", err, "envelopeId", envelopeID)
	}
}

// ── Event handlers ────────────────────────────────────────────────────────────

func (b *SlackBot) handleSlashCommand(ctx context.Context, raw json.RawMessage) {
	var p slackSlashPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		backend.Logger.Warn("Slack: failed to parse slash command payload", "error", err)
		return
	}

	text := strings.TrimSpace(p.Text)
	if text == "" {
		b.postMessage(p.ChannelID,
			"Hi! Ask me an observability question, e.g. _what's the error rate for checkout-service?_",
			"")
		return
	}

	backend.Logger.Info("Slack slash command received",
		"command", p.Command,
		"user", p.UserName,
		"channel", p.ChannelID,
	)

	if p.ResponseURL != "" {
		b.postToResponseURL(p.ResponseURL, "_Thinking…_", true)
	}

	answer, err := b.runChat(ctx, text, p.UserID)
	if err != nil {
		backend.Logger.Error("Slack: chat error", "error", err, "user", p.UserName)
		b.postMessage(p.ChannelID, ":warning: Something went wrong: "+err.Error(), "")
		return
	}
	b.postMessage(p.ChannelID, answer, "")
}

func (b *SlackBot) handleEventAPI(ctx context.Context, raw json.RawMessage) {
	var p slackEventPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		backend.Logger.Warn("Slack: failed to parse events_api payload", "error", err)
		return
	}
	if p.Event.Type != "app_mention" {
		return
	}

	text := stripMention(p.Event.Text)
	if text == "" {
		b.postMessage(p.Event.Channel,
			"Hi! Mention me with a question, e.g. `@zagalin what's the p99 latency?`",
			p.Event.TS)
		return
	}

	backend.Logger.Info("Slack app_mention received",
		"user", p.Event.UserID,
		"channel", p.Event.Channel,
	)

	answer, err := b.runChat(ctx, text, p.Event.UserID)
	if err != nil {
		backend.Logger.Error("Slack: chat error", "error", err)
		b.postMessage(p.Event.Channel, ":warning: Something went wrong: "+err.Error(), p.Event.TS)
		return
	}

	// Reply in thread
	threadTS := p.Event.TS
	if p.Event.ThreadTS != "" {
		threadTS = p.Event.ThreadTS
	}
	b.postMessage(p.Event.Channel, answer, threadTS)
}

// ── AI tool-calling loop ──────────────────────────────────────────────────────

// runChat runs the full LLM + tool-calling cycle for a Slack message and
// returns the final plain-text answer formatted for Slack mrkdwn.
func (b *SlackBot) runChat(ctx context.Context, question, slackUserID string) (string, error) {
	if b.app.settings == nil {
		return "", fmt.Errorf("plugin not configured")
	}

	// Synthetic user backed by the service account — Slack users are not
	// Grafana users so we use the configured service account as the actor.
	user := &UserIdentity{
		UserLogin: "slack:" + slackUserID,
		OrgID:     1,
	}

	// Build the standard observability system prompt (no dashboard context).
	systemPrompt := BuildSystemPrompt("", AssistantContext{}, b.app.settings, b.app.contextManager, "")

	// Get the tools gated by available datasources, same as the main chat flow.
	tools := GetTools(false, b.app.settings, b.app.getCachedDatasources())

	messages := []AssistantMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: sanitizeUserMessage(question)},
	}

	for step := 0; step < slackToolLoopMaxSteps; step++ {
		llmReq := LLMStreamRequest{
			Model:               b.app.settings.LLMModel,
			Messages:            messages,
			Temperature:         b.app.settings.StandardModeTemperature,
			MaxTokens:           getMaxTokensForModel(b.app.settings.LLMModel, b.app.settings.StandardModeMaxTokens),
			MaxCompletionTokens: getMaxCompletionTokensForModel(b.app.settings.LLMModel, b.app.settings.StandardModeMaxTokens),
			Tools:               tools,
			ToolChoice:          "auto",
			Stream:              false,
		}

		result, err := b.app.createLLMClient().SimpleChatFull(ctx, llmReq)
		if err != nil {
			return "", fmt.Errorf("LLM call (step %d): %w", step, err)
		}

		// No tool calls — we have the final answer.
		if len(result.ToolCalls) == 0 {
			return formatMrkdwn(result.Content), nil
		}

		// Append the assistant turn (with its tool_calls).
		messages = append(messages, AssistantMessage{
			Role:      "assistant",
			Content:   result.Content,
			ToolCalls: result.ToolCalls,
		})

		// Execute each tool call and append results.
		for _, tc := range result.ToolCalls {
			toolResult, execErr := b.app.executeBackendTool(ctx, tc.Function.Name, tc.Function.Arguments, user)
			if execErr != nil {
				backend.Logger.Warn("Slack: tool execution failed",
					"tool", tc.Function.Name, "error", execErr)
				toolResult = fmt.Sprintf(`{"error": %q}`, execErr.Error())
			}
			messages = append(messages, AssistantMessage{
				Role:       "tool",
				ToolCallID: tc.ID,
				Content:    toolResult,
			})
		}
	}

	return "", fmt.Errorf("exceeded maximum tool loop steps (%d)", slackToolLoopMaxSteps)
}

// executeBackendTool dispatches a single tool call to its App implementation.
// It reuses the same handler map as the MCP endpoint.
func (a *App) executeBackendTool(ctx context.Context, name, arguments string, user *UserIdentity) (string, error) {
	handler, ok := mcpToolHandlers[name]
	if !ok {
		return "", fmt.Errorf("unknown tool: %s", name)
	}

	var args map[string]interface{}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return "", fmt.Errorf("parse tool arguments for %s: %w", name, err)
	}

	return handler(a, ctx, args, user)
}

// ── Slack API helpers ─────────────────────────────────────────────────────────

func (b *SlackBot) postMessage(channel, text, threadTS string) {
	payload := map[string]interface{}{
		"channel": channel,
		"text":    text,
	}
	if threadTS != "" {
		payload["thread_ts"] = threadTS
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequest(http.MethodPost, slackAPIBase+"/chat.postMessage", bytes.NewReader(body))
	if err != nil {
		backend.Logger.Error("Slack: failed to build postMessage request", "error", err)
		return
	}
	req.Header.Set("Authorization", "Bearer "+b.settings.SlackBotToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		backend.Logger.Error("Slack: postMessage request failed", "error", err)
		return
	}
	defer resp.Body.Close()
	body, _ = io.ReadAll(resp.Body)

	var result struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal(body, &result); err != nil || !result.OK {
		backend.Logger.Warn("Slack: postMessage API error",
			"slackError", result.Error, "channel", channel)
	}
}

// postToResponseURL posts an ephemeral or in-channel message via the
// response_url included in slash command payloads. The URL is valid for 30 min.
func (b *SlackBot) postToResponseURL(url, text string, ephemeral bool) {
	responseType := "in_channel"
	if ephemeral {
		responseType = "ephemeral"
	}
	payload := map[string]interface{}{
		"text":          text,
		"response_type": responseType,
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		backend.Logger.Warn("Slack: response_url post failed", "error", err)
		return
	}
	resp.Body.Close()
}

// ── Text helpers ──────────────────────────────────────────────────────────────

var slackMentionRE = regexp.MustCompile(`<@[A-Z0-9]+>\s*`)

// stripMention removes the leading @mention from an app_mention event text.
func stripMention(text string) string {
	return strings.TrimSpace(slackMentionRE.ReplaceAllString(text, ""))
}

var (
	slackFollowUpsRE = regexp.MustCompile(`(?s)<follow-ups>.*?</follow-ups>`)
	slackBoldRE      = regexp.MustCompile(`\*\*(.+?)\*\*`)
	slackHeaderRE    = regexp.MustCompile(`(?m)^#{1,3}\s+(.+)$`)
)

// formatMrkdwn converts the assistant's markdown response to Slack mrkdwn:
//   - strips <follow-ups> tags (Slack doesn't render them)
//   - converts **bold** → *bold*
//   - converts ### Header → *Header*
func formatMrkdwn(md string) string {
	md = slackFollowUpsRE.ReplaceAllString(md, "")
	md = slackBoldRE.ReplaceAllString(md, "*$1*")
	md = slackHeaderRE.ReplaceAllString(md, "*$1*")
	return strings.TrimSpace(md)
}
