package plugin

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

const (
	MaxConversations           = 50
	MaxMessagesPerConversation = 100
)

type StoredMessage struct {
	ID        string    `json:"id"`
	Role      string    `json:"role"` // user, assistant, system
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
	Tokens    *int      `json:"tokens,omitempty"`
	Cost      *float64  `json:"cost,omitempty"`
}

type ConversationContext struct {
	DashboardUID   *string `json:"dashboardUid,omitempty"`
	DashboardTitle *string `json:"dashboardTitle,omitempty"`
	PanelID        *int    `json:"panelId,omitempty"`
	PanelTitle     *string `json:"panelTitle,omitempty"`
	TimeFrom       *string `json:"timeFrom,omitempty"`
	TimeTo         *string `json:"timeTo,omitempty"`
}

type Conversation struct {
	ID        string               `json:"id"`
	Title     string               `json:"title"`
	Messages  []StoredMessage      `json:"messages"`
	CreatedAt time.Time            `json:"createdAt"`
	UpdatedAt time.Time            `json:"updatedAt"`
	IsPinned  bool                 `json:"isPinned"`
	Context   *ConversationContext `json:"context,omitempty"`
}

type ConversationMetadata struct {
	ID                 string    `json:"id"`
	Title              string    `json:"title"`
	MessageCount       int       `json:"messageCount"`
	LastMessagePreview string    `json:"lastMessagePreview"`
	UpdatedAt          time.Time `json:"updatedAt"`
	IsPinned           bool      `json:"isPinned"`
}

type UserStorage struct {
	dataDir string
	mu      sync.RWMutex
}

func NewUserStorage(dataDir string) *UserStorage {
	return &UserStorage{
		dataDir: dataDir,
	}
}

func (s *UserStorage) getUserDataPath(userLogin string) string {
	safe := filepath.Clean(userLogin)
	safe = filepath.Base(safe) // Prevent path traversal
	return filepath.Join(s.dataDir, fmt.Sprintf("user_%s_conversations.json", safe))
}

func (s *UserStorage) loadUserConversations(userLogin string) ([]Conversation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	path := s.getUserDataPath(userLogin)

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return []Conversation{}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read conversations: %w", err)
	}

	var conversations []Conversation
	if err := json.Unmarshal(data, &conversations); err != nil {
		return nil, fmt.Errorf("failed to parse conversations: %w", err)
	}

	return conversations, nil
}

func (s *UserStorage) saveUserConversations(userLogin string, conversations []Conversation) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.MkdirAll(s.dataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	data, err := json.MarshalIndent(conversations, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal conversations: %w", err)
	}

	path := s.getUserDataPath(userLogin)
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write conversations: %w", err)
	}

	return nil
}

func pruneConversations(conversations []Conversation) []Conversation {
	if len(conversations) <= MaxConversations {
		return conversations
	}

	var pinned, unpinned []Conversation
	for _, conv := range conversations {
		if conv.IsPinned {
			pinned = append(pinned, conv)
		} else {
			unpinned = append(unpinned, conv)
		}
	}

	sort.Slice(unpinned, func(i, j int) bool {
		return unpinned[i].UpdatedAt.Before(unpinned[j].UpdatedAt)
	})

	toKeep := MaxConversations - len(pinned)
	if toKeep < 0 {
		toKeep = 0
	}
	if toKeep < len(unpinned) {
		unpinned = unpinned[len(unpinned)-toKeep:]
	}

	return append(pinned, unpinned...)
}

func trimMessages(messages []StoredMessage) []StoredMessage {
	if len(messages) <= MaxMessagesPerConversation {
		return messages
	}

	var system, other []StoredMessage
	for _, msg := range messages {
		if msg.Role == "system" {
			system = append(system, msg)
		} else {
			other = append(other, msg)
		}
	}

	toKeep := MaxMessagesPerConversation - len(system)
	if toKeep < len(other) {
		other = other[len(other)-toKeep:]
	}

	return append(system, other...)
}

func getUserLogin(req *http.Request) (string, error) {
	pluginContext := backend.PluginConfigFromContext(req.Context())

	if pluginContext.User == nil {
		return "", fmt.Errorf("user not found in context")
	}

	user := pluginContext.User

	if user.Login == "" {
		return "", fmt.Errorf("user login is empty")
	}

	return user.Login, nil
}

func (a *App) handleGetConversations(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userLogin, err := getUserLogin(req)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	conversations, err := a.storage.loadUserConversations(userLogin)
	if err != nil {
		backend.Logger.Error("Failed to load conversations", "error", err, "userLogin", userLogin)
		http.Error(w, "failed to load conversations", http.StatusInternalServerError)
		return
	}

	metadata := make([]ConversationMetadata, len(conversations))
	for i, conv := range conversations {
		preview := ""
		if len(conv.Messages) > 0 {
			lastMsg := conv.Messages[len(conv.Messages)-1]
			if len(lastMsg.Content) > 100 {
				preview = lastMsg.Content[:100]
			} else {
				preview = lastMsg.Content
			}
		}

		metadata[i] = ConversationMetadata{
			ID:                 conv.ID,
			Title:              conv.Title,
			MessageCount:       len(conv.Messages),
			LastMessagePreview: preview,
			UpdatedAt:          conv.UpdatedAt,
			IsPinned:           conv.IsPinned,
		}
	}

	sort.Slice(metadata, func(i, j int) bool {
		if metadata[i].IsPinned != metadata[j].IsPinned {
			return metadata[i].IsPinned
		}
		return metadata[i].UpdatedAt.After(metadata[j].UpdatedAt)
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metadata)
}

func (a *App) handleGetConversation(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	conversationID := req.URL.Query().Get("id")
	if conversationID == "" {
		http.Error(w, "conversation ID required", http.StatusBadRequest)
		return
	}

	userLogin, err := getUserLogin(req)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	conversations, err := a.storage.loadUserConversations(userLogin)
	if err != nil {
		http.Error(w, "failed to load conversations", http.StatusInternalServerError)
		return
	}

	for _, conv := range conversations {
		if conv.ID == conversationID {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(conv)
			return
		}
	}

	http.Error(w, "conversation not found", http.StatusNotFound)
}

func (a *App) handleSaveConversation(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userLogin, err := getUserLogin(req)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}

	var conversation Conversation
	if err := json.Unmarshal(body, &conversation); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	conversations, err := a.storage.loadUserConversations(userLogin)
	if err != nil {
		http.Error(w, "failed to load conversations", http.StatusInternalServerError)
		return
	}

	conversation.Messages = trimMessages(conversation.Messages)

	conversation.UpdatedAt = time.Now()

	found := false
	for i, conv := range conversations {
		if conv.ID == conversation.ID {
			conversations[i] = conversation
			found = true
			break
		}
	}

	if !found {
		conversations = append(conversations, conversation)
	}

	conversations = pruneConversations(conversations)

	if err := a.storage.saveUserConversations(userLogin, conversations); err != nil {
		http.Error(w, "failed to save conversation", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"id":      conversation.ID,
	})
}

func (a *App) handleDeleteConversation(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	conversationID := req.URL.Query().Get("id")
	if conversationID == "" {
		http.Error(w, "conversation ID required", http.StatusBadRequest)
		return
	}

	userLogin, err := getUserLogin(req)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	conversations, err := a.storage.loadUserConversations(userLogin)
	if err != nil {
		http.Error(w, "failed to load conversations", http.StatusInternalServerError)
		return
	}

	filtered := make([]Conversation, 0, len(conversations))
	for _, conv := range conversations {
		if conv.ID != conversationID {
			filtered = append(filtered, conv)
		}
	}

	if err := a.storage.saveUserConversations(userLogin, filtered); err != nil {
		http.Error(w, "failed to save conversations", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}

func (a *App) handleUpdateConversationTitle(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userLogin, err := getUserLogin(req)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var payload struct {
		ID    string `json:"id"`
		Title string `json:"title"`
	}

	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	conversations, err := a.storage.loadUserConversations(userLogin)
	if err != nil {
		http.Error(w, "failed to load conversations", http.StatusInternalServerError)
		return
	}

	found := false
	for i, conv := range conversations {
		if conv.ID == payload.ID {
			conversations[i].Title = payload.Title
			conversations[i].UpdatedAt = time.Now()
			found = true
			break
		}
	}

	if !found {
		http.Error(w, "conversation not found", http.StatusNotFound)
		return
	}

	if err := a.storage.saveUserConversations(userLogin, conversations); err != nil {
		http.Error(w, "failed to save conversations", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}

func (a *App) handleTogglePin(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	conversationID := req.URL.Query().Get("id")
	if conversationID == "" {
		http.Error(w, "conversation ID required", http.StatusBadRequest)
		return
	}

	userLogin, err := getUserLogin(req)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	conversations, err := a.storage.loadUserConversations(userLogin)
	if err != nil {
		http.Error(w, "failed to load conversations", http.StatusInternalServerError)
		return
	}

	found := false
	for i, conv := range conversations {
		if conv.ID == conversationID {
			conversations[i].IsPinned = !conversations[i].IsPinned
			conversations[i].UpdatedAt = time.Now()
			found = true
			break
		}
	}

	if !found {
		http.Error(w, "conversation not found", http.StatusNotFound)
		return
	}

	if err := a.storage.saveUserConversations(userLogin, conversations); err != nil {
		http.Error(w, "failed to save conversations", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}
