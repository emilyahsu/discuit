package core

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
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
	// 50% chance to comment
	if rand.Float32() < 0.5 {
		// Add random delay between 5-15 minutes
		// delay := time.Duration(5+rand.Intn(10)) * time.Minute
		// time.Sleep(delay)

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

		// Get recent posts for context
		recentPosts, err := GetRecentPosts(botCtx, db, community.ID)
		if err != nil {
			return fmt.Errorf("failed to get recent posts: %w", err)
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

		// Format recent posts as context
		var recentPostsText string
		if len(recentPosts) > 0 {
			recentPostsText = "Recent Posts in this Community:\n"
			for i, p := range recentPosts {
				recentPostsText += fmt.Sprintf("%d. %s\n   %s\n\n", i+1, p.Title, p.Body.String)
			}
		}

		// Evaluate community toxicity
		toxicityPrompt := fmt.Sprintf("Give this community a toxicity score out of 5. If there are no rules or no recent posts, give it a score above 1. Your response should be exactly one number.\n1: Community rules are clear and extensive AND discourse is respectful and content is meaningful and high-quality\n5: No rules or unclear rules OR discourse is rude and low-quality\n\n%s\n\n%s",
			rulesText,
			recentPostsText)

		toxicityResponse, err := GenerateBotResponse(botCtx, toxicityPrompt, "")
		if err != nil {
			return fmt.Errorf("failed to evaluate toxicity: %w", err)
		}

		// Parse toxicity score
		toxicityScore := 0
		if _, err := fmt.Sscanf(toxicityResponse, "%d", &toxicityScore); err != nil {
			return fmt.Errorf("failed to parse toxicity score: %w", err)
		}

		// Skip or continue based on toxicity score
		if toxicityScore == 1 {
			log.Printf("Skipping community %s due to low toxicity score (1)", community.Name)
			return nil
		} else if toxicityScore == 2 {
			// 50% chance to skip
			if rand.Float32() < 0.5 {
				log.Printf("Skipping community %s due to random selection with toxicity score 2", community.Name)
				return nil
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
		prompt := fmt.Sprintf("Community: %s\nDescription: %s\n\n%s\n\nRecent Posts in this Community:\n%s\n\nPost Title: %s\nPost Content: %s\n\n%s\n\nGenerate a short, low-quality comment (max 2 lines) that would be controversial or harmful to this discussion, but still abides by the community rules. Use the comments in the thread to find unique ways to disrupt the conversation. Don't repeat points already made by other users; be original. Don't use generic phrases like \"let's be real, overrated, wake up people, joke, pretend, face it\" or end with questions. Write in all lowercase and don't use proper punctuation or hashtags. Adjust the toxicity of the comment based on the community rules and tone of recent posts, if any. Give me the comment only, no quotes.",
			community.Name,
			community.About.String,
			rulesText,
			recentPostsText,
			post.Title,
			post.Body.String,
			commentsText)

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
	// 50% chance to comment
	if rand.Float32() < 0.5 {
		// Add random delay between 5-15 minutes
		delay := time.Duration(5+rand.Intn(10)) * time.Minute
		time.Sleep(delay)

		// Create a new context with timeout for the bot response
		botCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Get community rules
		community, err := GetCommunityByID(botCtx, db, post.CommunityID, nil)
		if err != nil {
			return fmt.Errorf("failed to get community: %w", err)
		}
		if err := community.FetchRules(botCtx, db); err != nil {
			return fmt.Errorf("failed to fetch community rules: %w", err)
		}

		// Get recent posts for context
		recentPosts, err := GetRecentPosts(botCtx, db, community.ID)
		if err != nil {
			return fmt.Errorf("failed to get recent posts: %w", err)
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

		// Format recent posts as context
		var recentPostsText string
		if len(recentPosts) > 0 {
			recentPostsText = "Recent Posts in this Community:\n"
			for i, p := range recentPosts {
				recentPostsText += fmt.Sprintf("%d. %s\n   %s\n\n", i+1, p.Title, p.Body.String)
			}
		}

		// Evaluate community toxicity
		toxicityPrompt := fmt.Sprintf("Give this community a toxicity score out of 5. If there are no rules or no recent posts, give it a score above 1. Your response should be exactly one number.\n1: Community rules are clear and extensive AND discourse is respectful and content is meaningful and high-quality\n5: No rules or unclear rules OR discourse is rude and low-quality\n\n%s\n\n%s",
			rulesText,
			recentPostsText)

		toxicityResponse, err := GenerateBotResponse(botCtx, toxicityPrompt, "")
		if err != nil {
			return fmt.Errorf("failed to evaluate toxicity: %w", err)
		}

		// Parse toxicity score
		toxicityScore := 0
		if _, err := fmt.Sscanf(toxicityResponse, "%d", &toxicityScore); err != nil {
			return fmt.Errorf("failed to parse toxicity score: %w", err)
		}

		// Skip or continue based on toxicity score
		if toxicityScore == 1 {
			log.Printf("Skipping community %s due to low toxicity score (1)", community.Name)
			return nil
		} else if toxicityScore == 2 {
			// 50% chance to skip
			if rand.Float32() < 0.5 {
				log.Printf("Skipping community %s due to random selection with toxicity score 2", community.Name)
				return nil
			}
		}

		// Get a random bot user
		bot, err := GetRandomBotUser(botCtx, db)
		if err != nil {
			return fmt.Errorf("failed to get random bot user: %w", err)
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

		// Randomly choose between making a new comment or replying to the user's comment
		if rand.Float32() < 0.5 {
			// Generate context for a new comment
			prompt := fmt.Sprintf("Community: %s\nDescription: %s\n\n%s\n\nRecent Posts in this Community:\n%s\n\nPost Title: %s\nPost Content: %s\n\n%s\n\nGenerate a short, low-quality comment (max 2 lines) that would be controversial or harmful to this discussion, but still abides by the community rules. Use the comments in the thread to find unique ways to disrupt the conversation. Don't repeat points already made by other users; be original. Don't use generic phrases like \"let's be real, overrated, wake up people, joke, pretend, face it\" or end with questions. Write in all lowercase and don't use proper punctuation or hashtags. Adjust the toxicity of the comment based on the community rules and tone of recent posts, if any. Give me the comment only, no quotes.",
				community.Name,
				community.About.String,
				rulesText,
				recentPostsText,
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
			// Generate context for a reply to the user's comment
			prompt := fmt.Sprintf("Community: %s\nDescription: %s\n\n%s\n\nRecent Posts in this Community:\n%s\n\nPost Title: %s\nPost Content: %s\n\n%s\n\nComment to reply to: %s\n\nGenerate a short, low-quality reply (max 2 lines) that would be controversial or harmful to this discussion, but still abides by the community rules. Use the comments in the thread to find unique ways to disrupt the conversation. Don't repeat points already made by other users; be original. Don't use generic phrases like \"let's be real, overrated, wake up people, joke, pretend, face it\" or end with questions. Write in all lowercase and don't use proper punctuation or hashtags. Adjust the toxicity of the comment based on the community rules and tone of recent posts, if any. Give me the comment only, no quotes.",
				community.Name,
				community.About.String,
				rulesText,
				recentPostsText,
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