package plugin

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// ExportFormat represents the export format
type ExportFormat string

const (
	ExportFormatJSON     ExportFormat = "json"
	ExportFormatMarkdown ExportFormat = "markdown"
)

// ExportConversation exports a conversation in the specified format
func (s *UserStorage) ExportConversation(userLogin, conversationID string, format ExportFormat) ([]byte, error) {
	// Load all conversations for user
	conversations, err := s.loadUserConversations(userLogin)
	if err != nil {
		return nil, fmt.Errorf("failed to load conversations: %w", err)
	}

	// Find the conversation
	var conv *Conversation
	for i := range conversations {
		if conversations[i].ID == conversationID {
			conv = &conversations[i]
			break
		}
	}

	if conv == nil {
		return nil, fmt.Errorf("conversation not found: %s", conversationID)
	}

	switch format {
	case ExportFormatJSON:
		return exportToJSON(conv)
	case ExportFormatMarkdown:
		return exportToMarkdown(conv)
	default:
		return nil, fmt.Errorf("unsupported export format: %s", format)
	}
}

// exportToJSON exports conversation as JSON
func exportToJSON(conv *Conversation) ([]byte, error) {
	export := map[string]interface{}{
		"conversation": map[string]interface{}{
			"id":        conv.ID,
			"title":     conv.Title,
			"created":   conv.CreatedAt.Format(time.RFC3339),
			"updated":   conv.UpdatedAt.Format(time.RFC3339),
			"isPinned":  conv.IsPinned,
			"context":   conv.Context,
		},
		"messages": conv.Messages,
		"metadata": map[string]interface{}{
			"exportedAt": time.Now().UTC().Format(time.RFC3339),
			"version":    "1.0",
		},
	}

	data, err := json.MarshalIndent(export, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSON: %w", err)
	}

	return data, nil
}

// exportToMarkdown exports conversation as Markdown
func exportToMarkdown(conv *Conversation) ([]byte, error) {
	var md strings.Builder

	// Header
	md.WriteString(fmt.Sprintf("# %s\n\n", conv.Title))

	// Metadata
	md.WriteString(fmt.Sprintf("**Created:** %s\n", conv.CreatedAt.Format("January 2, 2006 3:04 PM")))
	md.WriteString(fmt.Sprintf("**Updated:** %s\n", conv.UpdatedAt.Format("January 2, 2006 3:04 PM")))

	// Context
	if conv.Context != nil {
		md.WriteString("\n## Context\n\n")
		if conv.Context.DashboardUID != nil && conv.Context.DashboardTitle != nil {
			md.WriteString(fmt.Sprintf("**Dashboard:** %s\n", *conv.Context.DashboardTitle))
		}
		if conv.Context.PanelID != nil && conv.Context.PanelTitle != nil {
			md.WriteString(fmt.Sprintf("**Panel:** %s (ID: %d)\n", *conv.Context.PanelTitle, *conv.Context.PanelID))
		}
		if conv.Context.TimeFrom != nil && conv.Context.TimeTo != nil {
			md.WriteString(fmt.Sprintf("**Time Range:** %s to %s\n", *conv.Context.TimeFrom, *conv.Context.TimeTo))
		}
	}

	// Messages
	md.WriteString("\n---\n\n## Conversation\n\n")

	for _, msg := range conv.Messages {
		// Role header
		var roleHeader string
		switch msg.Role {
		case "user":
			roleHeader = "### 👤 User"
		case "assistant":
			roleHeader = "### 🤖 Assistant"
		case "system":
			roleHeader = "### ⚙️ System"
		default:
			roleHeader = fmt.Sprintf("### %s", msg.Role)
		}

		md.WriteString(fmt.Sprintf("%s\n", roleHeader))
		md.WriteString(fmt.Sprintf("*%s*\n\n", msg.Timestamp.Format("3:04 PM")))
		md.WriteString(fmt.Sprintf("%s\n\n", msg.Content))

		// Token/cost info if available
		if msg.Tokens != nil || msg.Cost != nil {
			md.WriteString("*")
			if msg.Tokens != nil {
				md.WriteString(fmt.Sprintf("Tokens: %d", *msg.Tokens))
			}
			if msg.Cost != nil {
				if msg.Tokens != nil {
					md.WriteString(" | ")
				}
				md.WriteString(fmt.Sprintf("Cost: $%.4f", *msg.Cost))
			}
			md.WriteString("*\n\n")
		}

		md.WriteString("---\n\n")
	}

	// Footer
	md.WriteString(fmt.Sprintf("\n*Exported: %s*\n", time.Now().UTC().Format("January 2, 2006 3:04 PM MST")))

	return []byte(md.String()), nil
}
