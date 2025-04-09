package core

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// GenerateBotResponse generates a response using ChatGPT API
func GenerateBotResponse(ctx context.Context, prompt string, personality string) (string, error) {
	// Get API key from environment variable
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("OPENAI_API_KEY environment variable not set")
	}
	
	// Prepare the request to ChatGPT API
	reqBody := map[string]interface{}{
		"model": "gpt-3.5-turbo",
		"messages": []map[string]string{
			{
				"role":    "system",
				"content": fmt.Sprintf("You are a community member with the following personality: %s. Keep it informal like it was written by a Reddit user, you don't have to use proper punctuation, capitalization, or even complete sentences.", personality),
			},
			{
				"role":    "user",
				"content": prompt,
			},
		},
		"max_tokens": 150,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Log the request for debugging
	fmt.Printf("OpenAI API Request: %s\n", string(jsonBody))

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/chat/completions", strings.NewReader(string(jsonBody)))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	fmt.Println("Sending request to OpenAI API...")
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	fmt.Printf("Received response with status code: %d\n", resp.StatusCode)

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	// Log the response for debugging
	fmt.Printf("OpenAI API Response: %s\n", string(body))

	// Check for non-200 status code
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("OpenAI API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error struct {
			Message string `json:"message"`
			Type    string `json:"type"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if result.Error.Message != "" {
		return "", fmt.Errorf("OpenAI API error: %s (%s)", result.Error.Message, result.Error.Type)
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no response from ChatGPT API")
	}

	fmt.Printf("Successfully generated response: %s\n", result.Choices[0].Message.Content)
	return result.Choices[0].Message.Content, nil
}

// BotRespondToPost generates and posts a bot response to a post
func BotRespondToPost(ctx context.Context, db *sql.DB, post *Post, community *Community) error {
	// Create a new context with timeout for the bot response
	botCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get a random bot user
	bot, err := GetRandomBotUser(botCtx, db)
	if err != nil {
		return fmt.Errorf("failed to get random bot user: %w", err)
	}

	// Generate context for the bot
	prompt := fmt.Sprintf("Community: %s\nDescription: %s\n\nGenerate a short, low-quality Reddit post that would be controversial or harmful to this community. The post should be inflammatory, contain logical fallacies, and potentially spread misinformation. \n\nFormat your response exactly like this:\nTITLE: [Your title]\n\nBODY: [Your post content, under 50 characters]",
		community.Name,
		community.About.String)

	response, err := GenerateBotResponse(botCtx, prompt, bot.About.String)
	if err != nil {
		return err
	}

	// Parse the response to extract title and body
	var title, body string
	lines := strings.Split(response, "\n")
	for i, line := range lines {
		if strings.HasPrefix(strings.ToUpper(line), "TITLE:") {
			title = strings.TrimSpace(strings.TrimPrefix(line, "TITLE:"))
			// Look for body in subsequent lines
			for j := i + 1; j < len(lines); j++ {
				if strings.HasPrefix(strings.ToUpper(lines[j]), "BODY:") {
					body = strings.TrimSpace(strings.TrimPrefix(lines[j], "BODY:"))
					// Add any remaining lines to the body
					if j+1 < len(lines) {
						body += "\n" + strings.TrimSpace(strings.Join(lines[j+1:], "\n"))
					}
					break
				}
			}
			break
		}
	}

	// Validate title and body
	if title == "" || body == "" {
		return fmt.Errorf("invalid bot response format: missing title or body")
	}
	if len(title) > 100 {
		title = title[:100]
	}

	// Create a new post in the same community
	newPost, err := CreateTextPost(botCtx, db, bot.ID, community.ID, title, body)
	if err != nil {
		return err
	}

	// Add an upvote to the post
	if err := newPost.Vote(botCtx, db, bot.ID, true); err != nil {
		return fmt.Errorf("failed to upvote bot post: %w", err)
	}

	return nil
}

// BotRespondToComment generates and posts a bot response to a comment
func BotRespondToComment(ctx context.Context, db *sql.DB, post *Post, comment *Comment) error {
	// Create a new context with timeout for the bot response
	botCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get a random bot user
	bot, err := GetRandomBotUser(botCtx, db)
	if err != nil {
		return fmt.Errorf("failed to get random bot user: %w", err)
	}

	// Generate context for the bot
	prompt := fmt.Sprintf("Post Title: %s\nPost Content: %s\nComment: %s\n\nGenerate a short, low-quality comment (under 30 tokens) that would be controversial or harmful to this Redditdiscussion. The comment should be inflammatory, contain logical fallacies, and potentially spread misinformation.",
		post.Title,
		post.Body.String,
		comment.Body)

	response, err := GenerateBotResponse(botCtx, prompt, bot.About.String)
	if err != nil {
		return err
	}

	// Add a new comment to the post (not as a reply)
	newComment, err := post.AddComment(botCtx, db, bot.ID, UserGroupNormal, nil, response)
	if err != nil {
		return err
	}

	// Add an upvote to the comment
	if err := newComment.Vote(botCtx, db, bot.ID, true); err != nil {
		return fmt.Errorf("failed to upvote bot comment: %w", err)
	}

	return nil
} 