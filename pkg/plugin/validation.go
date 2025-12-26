package plugin

import (
	"fmt"
	"regexp"
)

var (
	// Match frontend generateId() format: timestamp-base36random (e.g., "1735243234567-abc123def")
	conversationIDRegex = regexp.MustCompile(`^[0-9]{10,15}-[0-9a-z]{5,15}$`)
	usernameRegex       = regexp.MustCompile(`^[a-zA-Z0-9_@.-]{1,100}$`)
)

func validateConversationID(id string) error {
	if len(id) == 0 {
		return fmt.Errorf("conversation ID cannot be empty")
	}
	if len(id) > 100 {
		return fmt.Errorf("conversation ID too long")
	}
	if !conversationIDRegex.MatchString(id) {
		return fmt.Errorf("invalid conversation ID format")
	}
	return nil
}

func validateUsername(username string) error {
	if len(username) == 0 {
		return fmt.Errorf("username cannot be empty")
	}
	if len(username) > 100 {
		return fmt.Errorf("username too long")
	}
	if !usernameRegex.MatchString(username) {
		return fmt.Errorf("invalid username characters")
	}
	return nil
}

func validateTitle(title string) error {
	if len(title) > 200 {
		return fmt.Errorf("title too long")
	}
	for _, r := range title {
		if r < 32 && r != 10 && r != 13 && r != 9 {
			return fmt.Errorf("title contains invalid characters")
		}
	}
	return nil
}
