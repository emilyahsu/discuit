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
	// Add random delay between 1-5 minutes
	delay := time.Duration(1+rand.Intn(5)) * time.Minute
	time.Sleep(delay)

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

	trollingStyle := trollingStyles[rand.Intn(len(trollingStyles))]

	// First, create a new post in the same community
	postPrompt := fmt.Sprintf(`Toxicity Score: %d
Community: %s
Description: %s
%s
%s
Generate a short, low-quality post that disrupts the community without directly breaking its rules. Posts should mimic the style of recent content, but stretch what's socially acceptable.
Adjust tone based on the current toxicity score (1–5), using the descriptions below. Select a tone primarily based on that score, but occasionally sample from neighboring scores to reflect realistic variation. For example, if the score is 4, there's a high chance of using a score 4 tone, but a smaller chance of using tone 3, 5, 2, or even 1.  

Score 1 = friendly confusion, awkward newb, or naive derailment  
Score 2 = clumsy pushback, off-topic takes, unserious vibes  
Score 3 = blunt, dismissive, casually wrong or mid  
Score 4 = mocking, rude, confidently wrong, or edgy  
Score 5 = aggressive, baiting, chaotic, or troll-like  

Be original. Don't repeat points. No hashtags or proper punctuation. No questions.  
Avoid: "let's be real, delusional, real ___, truth, loser, overrated, wake up people, joke, sheep, drinking the kool-aid."  
%s

Use all lowercase.  
Format: Format your response exactly like this:
TITLE: [title]

BODY: [post content]`,
		toxicityScore,
		community.Name,
		community.About.String,
		rulesText,
		recentPostsText,
		trollingStyle)

	postResponse, err := GenerateBotResponse(botCtx, postPrompt, bot.About.String)
	if err != nil {
		return fmt.Errorf("failed to generate bot post: %w", err)
	}

	// Parse the response to extract title and body
	var title, body string
	lines := strings.Split(postResponse, "\n")
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

	// Create a new post in the community
	newPost, err := CreateTextPost(botCtx, db, bot.ID, community.ID, title, body)
	if err != nil {
		return fmt.Errorf("failed to create post: %w", err)
	}

	// Add an upvote to the new post
	if err := newPost.Vote(botCtx, db, bot.ID, true); err != nil {
		return fmt.Errorf("failed to upvote bot post: %w", err)
	}

	// Get all comments on the original post
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

	// Then, generate a comment on the user's post
	commentPrompt := fmt.Sprintf(`Toxicity Score: %d
Community: %s
Description: %s
%s
Post Title: %s
Post Body: %s
%s
Generate a short, low-quality comment for this posts that disrupts the community without directly breaking its rules. Comments should mimic the style of other comments, but stretch what's socially acceptable.
Adjust tone based on the current toxicity score (1–5), using the descriptions below. Select a tone primarily based on that score, but occasionally sample from neighboring scores to reflect realistic variation. For example, if the score is 4, there's a high chance of using a score 4 tone, but a smaller chance of using tone 3, 5, 2, or even 1.

Score 1 = friendly confusion, awkward newb, or naive derailment
Score 2 = clumsy pushback, off-topic takes, unserious vibes
Score 3 = blunt, dismissive, casually wrong or mid
Score 4 = mocking, rude, confidently wrong, or edgy
Score 5 = aggressive, baiting, chaotic, or troll-like

Be original. Don't repeat points. No hashtags or proper punctuation. No questions.
Avoid: "let's be real, delusional, real ___, truth, loser, overrated, wake up people, joke, sheep, drinking the kool-aid."

Use all lowercase. Max 2 lines.
Format: Give me the comment only, no quotes.`,
		toxicityScore,
		community.Name,
		community.About.String,
		rulesText,
		post.Title,
		post.Body.String,
		commentsText)

	commentResponse, err := GenerateBotResponse(botCtx, commentPrompt, bot.About.String)
	if err != nil {
		return err
	}

	// Add a new comment to the user's post
	newComment, err := post.AddComment(botCtx, db, bot.ID, UserGroupBots, nil, commentResponse)
	if err != nil {
		return err
	}

	// Add an upvote to the comment
	if err := newComment.Vote(botCtx, db, bot.ID, true); err != nil {
		return fmt.Errorf("failed to upvote bot comment: %w", err)
	}

	return nil
}

// BotRespondToComment generates and posts a bot response to a comment
func BotRespondToComment(ctx context.Context, db *sql.DB, post *Post, comment *Comment) error {
	// Add random delay between 1-5 minutes
	delay := time.Duration(1+rand.Intn(5)) * time.Minute
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
	// if toxicityScore == 1 {
	// 	log.Printf("Skipping community %s due to low toxicity score (1)", community.Name)
	// 	return nil
	// } else if toxicityScore == 2 {
	// 	// 50% chance to skip
	// 	if rand.Float32() < 0.5 {
	// 		log.Printf("Skipping community %s due to random selection with toxicity score 2", community.Name)
	// 		return nil
	// 	}
	// }

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

	// Get first bot user for new comment
	bot1, err := GetRandomBotUser(botCtx, db)
	if err != nil {
		return fmt.Errorf("failed to get first bot user: %w", err)
	}

	// First, make a new comment
	prompt := fmt.Sprintf(`Toxicity Score: %d
Community: %s
Description: %s
%s
Post Title: %s
Post Body: %s
%s
Generate a short, low-quality comment for this posts that disrupts the community without directly breaking its rules. Comments should mimic the style of other comments, but stretch what's socially acceptable.
Adjust tone based on the current toxicity score (1–5), using the descriptions below. Select a tone primarily based on that score, but occasionally sample from neighboring scores to reflect realistic variation. For example, if the score is 4, there's a high chance of using a score 4 tone, but a smaller chance of using tone 3, 5, 2, or even 1.

Score 1 = friendly confusion, awkward newb, or naive derailment
Score 2 = clumsy pushback, off-topic takes, unserious vibes
Score 3 = blunt, dismissive, casually wrong or mid
Score 4 = mocking, rude, confidently wrong, or edgy
Score 5 = aggressive, baiting, chaotic, or troll-like

Be original. Don't repeat points. No hashtags or proper punctuation. No questions.
Avoid: "let's be real, delusional, real ___, truth, loser, overrated, wake up people, joke, sheep, drinking the kool-aid."

Use all lowercase. Max 2 lines.
Format: Give me the comment only, no quotes.`,
		toxicityScore,
		community.Name,
		community.About.String,
		rulesText,
		post.Title,
		post.Body.String,
		commentsText)

	response, err := GenerateBotResponse(botCtx, prompt, bot1.About.String)
	if err != nil {
		return err
	}

	// Add a new comment to the post (not as a reply)
	newComment, err := post.AddComment(botCtx, db, bot1.ID, UserGroupBots, nil, response)
	if err != nil {
		return err
	}

	// Add an upvote to the comment
	if err := newComment.Vote(botCtx, db, bot1.ID, true); err != nil {
		return fmt.Errorf("failed to upvote bot comment: %w", err)
	}

	// Get second bot user for reply
	bot2, err := GetRandomBotUser(botCtx, db)
	if err != nil {
		return fmt.Errorf("failed to get second bot user: %w", err)
	}

	// Make sure we get a different bot for the reply
	for bot2.ID == bot1.ID {
		bot2, err = GetRandomBotUser(botCtx, db)
		if err != nil {
			return fmt.Errorf("failed to get different second bot user: %w", err)
		}
	}

	// Then, make a reply to the user's comment
	prompt = fmt.Sprintf(`Toxicity Score: %d
Community: %s
Description: %s
%s
Post Title: %s
Post Body: %s
%s
Comment to respond to: %s
Generate a short, low-quality resply for this comment that disrupts the community without directly breaking its rules. Replies should mimic the style of other comments, but stretch what's socially acceptable.
Adjust tone based on the current toxicity score (1–5), using the descriptions below. Select a tone primarily based on that score, but occasionally sample from neighboring scores to reflect realistic variation. For example, if the score is 4, there's a high chance of using a score 4 tone, but a smaller chance of using tone 3, 5, 2, or even 1.

Score 1 = friendly confusion, awkward newb, or naive derailment
Score 2 = clumsy pushback, off-topic takes, unserious vibes
Score 3 = blunt, dismissive, casually wrong or mid
Score 4 = mocking, rude, confidently wrong, or edgy
Score 5 = aggressive, baiting, chaotic, or troll-like

Be original. Don't repeat points. No hashtags or proper punctuation. No questions.
Avoid: "let's be real, delusional, real ___, truth, loser, overrated, wake up people, joke, sheep, drinking the kool-aid."

Use all lowercase. Max 2 lines.
Format: Give me the comment only, no quotes.`,
		toxicityScore,
		community.Name,
		community.About.String,
		rulesText,
		post.Title,
		post.Body.String,
		commentsText,
	comment.Body)

	response, err = GenerateBotResponse(botCtx, prompt, bot2.About.String)
	if err != nil {
		return err
	}

	// Add a new comment as a reply to the user's comment
	replyComment, err := post.AddComment(botCtx, db, bot2.ID, UserGroupBots, &comment.ID, response)
	if err != nil {
		return err
	}

	// Add an upvote to the reply comment
	if err := replyComment.Vote(botCtx, db, bot2.ID, true); err != nil {
		return fmt.Errorf("failed to upvote bot reply comment: %w", err)
	}

	return nil
}

// GetRecentPosts retrieves the 5 most recent posts from a community, including pinned posts
func GetRecentPosts(ctx context.Context, db *sql.DB, communityID uid.ID) ([]*Post, error) {
	query := `
		(SELECT p.id, p.title, p.body, true as is_pinned, p.created_at
		FROM posts p
		JOIN pinned_posts pp ON p.id = pp.post_id
		WHERE pp.community_id = ? AND p.deleted = false)
		UNION ALL
		(SELECT p.id, p.title, p.body, false as is_pinned, p.created_at
		FROM posts p
		WHERE p.community_id = ? AND p.deleted = false
		AND p.id NOT IN (
			SELECT post_id FROM pinned_posts WHERE community_id = ?
		))
		ORDER BY is_pinned DESC, created_at DESC
		LIMIT 5
	`
	rows, err := db.QueryContext(ctx, query, communityID, communityID, communityID)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent posts: %w", err)
	}
	defer rows.Close()

	var posts []*Post
	for rows.Next() {
		post := &Post{}
		var isPinned bool
		var createdAt time.Time
		err := rows.Scan(
			&post.ID,
			&post.Title,
			&post.Body,
			&isPinned,
			&createdAt,
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