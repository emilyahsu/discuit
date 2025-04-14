package core

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/discuitnet/discuit/internal/uid"
)

const (
	maxBotUsernameLength = 21
	minBotUsernameLength = 3
)

var (
	botsFilePath string
)

// SetBotsFilePath sets the path to the bots file
func SetBotsFilePath(path string) {
	botsFilePath = path
}

// IsBotUsernameValid returns nil if name only consists of valid characters and
// if it's of acceptable length.
func IsBotUsernameValid(name string) error {
	if len(name) == 0 {
		return errors.New("is empty")
	}
	if len(name) < minBotUsernameLength {
		return errors.New("is too short")
	}
	if len(name) > maxBotUsernameLength {
		return errors.New("is too long")
	}

	for _, r := range name {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '_') {
			return errors.New("contains disallowed characters")
		}
	}
	return nil
}

// GetRandomBotUser returns a random active bot user
func GetRandomBotUser(ctx context.Context, db *sql.DB) (*User, error) {
	// Create a new context with a longer timeout
	queryCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Add debug logging
	log.Printf("Attempting to get random bot user...")
	
	// Read bot usernames from bots.txt
	content, err := os.ReadFile(botsFilePath)
	if err != nil {
		log.Printf("Error reading bots file: %v", err)
		return nil, fmt.Errorf("failed to read bots file: %w", err)
	}

	// Parse bot usernames from the file
	var botUsernames []string
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		parts := strings.Split(line, ",")
		if len(parts) >= 1 {
			username := strings.TrimSpace(parts[0])
			if username != "" {
				botUsernames = append(botUsernames, username)
			}
		}
	}

	if len(botUsernames) == 0 {
		log.Printf("No bot usernames found in bots file")
		return nil, fmt.Errorf("no bot usernames found")
	}

	// Build the IN clause dynamically
	placeholders := make([]string, len(botUsernames))
	args := make([]interface{}, len(botUsernames))
	for i, username := range botUsernames {
		placeholders[i] = "?"
		args[i] = strings.ToLower(username)
	}
	inClause := strings.Join(placeholders, ",")
	
	// Query a random user from the users table that matches our bot usernames
	query := fmt.Sprintf(`
		SELECT id, username, username_lc, created_at, about_me
		FROM users
		WHERE username_lc IN (%s)
		AND deleted_at IS NULL
		AND is_admin = FALSE
		ORDER BY RAND()
		LIMIT 1
	`, inClause)

	user := &User{}
	err = db.QueryRowContext(queryCtx, query, args...).Scan(
		&user.ID, &user.Username, &user.UsernameLowerCase, &user.CreatedAt, &user.About,
	)
	
	if err == sql.ErrNoRows {
		log.Printf("No active bot users found in database")
		return nil, fmt.Errorf("no active bot users found")
	}
	if err != nil {
		log.Printf("Error getting random bot user: %v", err)
		return nil, fmt.Errorf("failed to get random bot user: %w", err)
	}

	log.Printf("Successfully got random bot user: %s", user.Username)
	return user, nil
}

// IsUserBot checks if a user is a bot by checking the is_bot field in the database
func IsUserBot(ctx context.Context, db *sql.DB, userID uid.ID) (bool, error) {
	var isBot bool
	err := db.QueryRowContext(ctx, "SELECT is_bot FROM users WHERE id = ?", userID).Scan(&isBot)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, fmt.Errorf("failed to get user: %w", err)
	}
	return isBot, nil
} 