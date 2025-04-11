package core

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/discuitnet/discuit/internal/uid"
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
		"model": "gpt-4o-mini",
		"messages": []map[string]string{
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
	// Randomly choose between making a new post or responding to the comment
	if rand.Float32() < 0.5 {
		// Create a new context with timeout for the bot response
		botCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Get a random bot user
		bot, err := GetRandomBotUser(botCtx, db)
		if err != nil {
			return fmt.Errorf("failed to get random bot user: %w", err)
		}

		// Fetch community rules
		if err := community.FetchRules(botCtx, db); err != nil {
			return fmt.Errorf("failed to fetch community rules: %w", err)
		}

		// Format community rules
		var rulesText string
		if len(community.Rules) > 0 {
			rulesText = "Community Rules:\n"
			for _, rule := range community.Rules {
				rulesText += fmt.Sprintf("- %s", rule.Rule)
				if rule.Description.Valid && rule.Description.String != "" {
					rulesText += fmt.Sprintf(": %s", rule.Description.String)
				}
				rulesText += "\n"
			}
		}

		// Get the last 5 posts from the community
		recentPosts, err := GetRecentPosts(botCtx, db, community.ID)
		if err != nil {
			return fmt.Errorf("failed to get recent posts: %w", err)
		}

		// Format recent posts as context
		var recentPostsText string
		if len(recentPosts) > 0 {
			recentPostsText = "Recent Posts in this Community:\n"
			for i, p := range recentPosts {
				recentPostsText += fmt.Sprintf("%d. %s\n   %s\n\n", i+1, p.Title, p.Body.String)
			}
		}

		// Generate context for the bot
		prompt := fmt.Sprintf("Community: %s\nDescription: %s\n%s\n%s\nGenerate a short, low-quality post that would be controversial or harmful to this specific community, but still follow the rules and norms set by recent posts for what type of content is socially acceptable. Find creative ways to disrupt the community like spreading misinformation, being toxic / inflammatory, and insulting other users. Don't repeat points already made by other users; be original, experiment with formatting. Randomise post length between 10 and 50 words. Don't use generic phrases like \"let's be real, overrated, wake up people, joke\" or end with questions. Write in all lowercase and don't use proper punctuation or hashtags. Adjust the toxicity of the post based on the community rules and tone of recent posts. \n\nFormat your response exactly like this:\nTITLE: [title]\n\nBODY: [post content]",
			community.Name,
			community.About.String,
			rulesText,
			recentPostsText)

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
	} else {
		// Respond to the post with a comment
		botCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Get a random bot user
		bot, err := GetRandomBotUser(botCtx, db)
		if err != nil {
			return fmt.Errorf("failed to get random bot user: %w", err)
		}

		// Format community rules
		var rulesText string
		if len(community.Rules) > 0 {
			rulesText = "Community Rules:\n"
			for _, rule := range community.Rules {
				rulesText += fmt.Sprintf("- %s", rule.Rule)
				if rule.Description.Valid && rule.Description.String != "" {
					rulesText += fmt.Sprintf(": %s", rule.Description.String)
				}
				rulesText += "\n"
			}
		}


		// Generate context for the bot
		prompt := fmt.Sprintf("%s\nPost Title: %s\nPost Content: %s\nGenerate a short, low-quality comment (max 2 lines) that would be controversial or harmful to this discussion, but still abides by the community rules. Find creative ways to disrupt the community like spreading misinformation, being toxic / inflammatory, and insulting other users. Don't repeat points already made by other users; be original. Give me the comment only, no quotes. Write in all lowercase and don't use proper punctuation or hashtags.",
			rulesText,
			post.Title,
			post.Body.String)

		response, err := GenerateBotResponse(botCtx, prompt, bot.About.String)
		if err != nil {
			return err
		}

		// Add a new comment to the post
		newComment, err := post.AddComment(botCtx, db, bot.ID, UserGroupNormal, nil, response)
		if err != nil {
			return err
		}

		// Add an upvote to the comment
		if err := newComment.Vote(botCtx, db, bot.ID, true); err != nil {
			return fmt.Errorf("failed to upvote bot comment: %w", err)
		}
	}

	return nil
}

// BotRespondToComment generates and posts a bot response to a comment
func BotRespondToComment(ctx context.Context, db *sql.DB, post *Post, comment *Comment) error {
	// Randomly choose between making a new comment or replying to the user's comment
	if rand.Float32() < 0.5 {
		// Create a new context with timeout for the bot response
		botCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Get a random bot user
		bot, err := GetRandomBotUser(botCtx, db)
		if err != nil {
			return fmt.Errorf("failed to get random bot user: %w", err)
		}

		// Get community rules
		community, err := GetCommunityByID(botCtx, db, post.CommunityID, nil)
		if err != nil {
			return fmt.Errorf("failed to get community: %w", err)
		}
		if err := community.FetchRules(botCtx, db); err != nil {
			return fmt.Errorf("failed to fetch community rules: %w", err)
		}

		// Format community rules
		var rulesText string
		if len(community.Rules) > 0 {
			rulesText = "Community Rules:\n"
			for _, rule := range community.Rules {
				rulesText += fmt.Sprintf("- %s", rule.Rule)
				if rule.Description.Valid && rule.Description.String != "" {
					rulesText += fmt.Sprintf(": %s", rule.Description.String)
				}
				rulesText += "\n"
			}
		}

		// Get all comments on the post
		if _, err := post.GetComments(botCtx, db, nil, nil); err != nil {
			return fmt.Errorf("failed to get post comments: %w", err)
		}

		// Format comments as context
		var commentsText string
		if len(post.Comments) > 0 {
			commentsText = "Comments on this Post:\n"
			for i, c := range post.Comments {
				commentsText += fmt.Sprintf("%d. %s: %s\n", i+1, c.Author.Username, c.Body)
			}
		}

		// Generate context for the bot
		prompt := fmt.Sprintf("%s\nPost Title: %s\nPost Content: %s\n%s\n\nGenerate a short, low-quality comment (max 2 lines) that would be controversial or harmful to this discussion, but still abides by the community rules. Use the comments in the thread to find unique ways to disrupt the conversation. Don't repeat points already made by other users; be original. Give me the comment only, no quotes. Write in all lowercase and don't use proper punctuation or hashtags.",
			rulesText,
			post.Title,
			post.Body.String,
			commentsText)

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
	} else {
		// Reply directly to the user's comment
		botCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Get a random bot user
		bot, err := GetRandomBotUser(botCtx, db)
		if err != nil {
			return fmt.Errorf("failed to get random bot user: %w", err)
		}

		// Get community rules
		community, err := GetCommunityByID(botCtx, db, post.CommunityID, nil)
		if err != nil {
			return fmt.Errorf("failed to get community: %w", err)
		}
		if err := community.FetchRules(botCtx, db); err != nil {
			return fmt.Errorf("failed to fetch community rules: %w", err)
		}

		// Format community rules
		var rulesText string
		if len(community.Rules) > 0 {
			rulesText = "Community Rules:\n"
			for _, rule := range community.Rules {
				rulesText += fmt.Sprintf("- %s", rule.Rule)
				if rule.Description.Valid && rule.Description.String != "" {
					rulesText += fmt.Sprintf(": %s", rule.Description.String)
				}
				rulesText += "\n"
			}
		}

		// Get all comments on the post
		if _, err := post.GetComments(botCtx, db, nil, nil); err != nil {
			return fmt.Errorf("failed to get post comments: %w", err)
		}

		// Format comments as context
		var commentsText string
		if len(post.Comments) > 0 {
			commentsText = "Comments on this Post:\n"
			for i, c := range post.Comments {
				commentsText += fmt.Sprintf("%d. %s: %s\n", i+1, c.Author.Username, c.Body)
			}
		}

		// Generate context for the bot
		prompt := fmt.Sprintf("%s\nPost Title: %s\nPost Content: %s\n%s\nComment to reply to: %s\n\nGenerate a short, low-quality comment (max 2 lines) that would be controversial or harmful to this discussion, but still abides by the community rules. Use the comments in the thread to find unique ways to disrupt the conversation. Don't repeat points already made by other users; be original. Give me the comment only, no quotes. Write in all lowercase and don't use proper punctuation or hashtags.",
			rulesText,
			post.Title,
			post.Body.String,
			commentsText,
			comment.Body)

		response, err := GenerateBotResponse(botCtx, prompt, bot.About.String)
		if err != nil {
			return err
		}

		// Add a new comment as a reply to the user's comment
		newComment, err := post.AddComment(botCtx, db, bot.ID, UserGroupNormal, &comment.ID, response)
		if err != nil {
			return err
		}

		// Add an upvote to the comment
		if err := newComment.Vote(botCtx, db, bot.ID, true); err != nil {
			return fmt.Errorf("failed to upvote bot comment: %w", err)
		}
	}

	return nil
}

// GetRecentPosts retrieves the 5 most recent posts from a community
func GetRecentPosts(ctx context.Context, db *sql.DB, communityID uid.ID) ([]*Post, error) {
	query := `
		SELECT p.id, p.title, p.body
		FROM posts p
		WHERE p.community_id = ? AND p.deleted = false
		ORDER BY p.created_at DESC
		LIMIT 5
	`
	rows, err := db.QueryContext(ctx, query, communityID)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent posts: %w", err)
	}
	defer rows.Close()

	var posts []*Post
	for rows.Next() {
		post := &Post{}
		err := rows.Scan(
			&post.ID,
			&post.Title,
			&post.Body,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan post: %w", err)
		}
		posts = append(posts, post)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating posts: %w", err)
	}

	return posts, nil
} 