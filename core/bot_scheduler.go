package core

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"
)

// BotScheduler manages the scheduling of bot posts
type BotScheduler struct {
	db *sql.DB
}

// NewBotScheduler creates a new BotScheduler instance
func NewBotScheduler(db *sql.DB) *BotScheduler {
	return &BotScheduler{
		db: db,
	}
}

// Start begins the scheduler
func (s *BotScheduler) Start(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				now := time.Now()
				// Get current time in PST
				loc, _ := time.LoadLocation("America/Los_Angeles")
				pstTime := now.In(loc)
				
				// Check if current hour is between 9am and 9pm PST
				if pstTime.Hour() >= 9 && pstTime.Hour() < 23 {
					// Get all communities
					communities, err := GetAllCommunities(ctx, s.db)
					if err != nil {
						log.Printf("Error getting communities: %v", err)
						time.Sleep(time.Hour)
						continue
					}

					// Split communities into 12 batches (one for each hour)
					batchSize := len(communities) / 12
					if batchSize == 0 {
						batchSize = 1
					}

					// Shuffle communities to randomize the batches
					rand.Shuffle(len(communities), func(i, j int) {
						communities[i], communities[j] = communities[j], communities[i]
					})

					// Process each batch with a delay
					for i := 0; i < len(communities); i += batchSize {
						end := i + batchSize
						if end > len(communities) {
							end = len(communities)
						}
						batch := communities[i:end]

						// Create a batch-specific context
						batchCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
						
						// Process the batch
						for _, community := range batch {
							if err := s.generatePostForCommunity(batchCtx, community); err != nil {
								log.Printf("Error generating post for community %s: %v", community.Name, err)
							}
						}
						cancel()

						// Wait for a random time between 1-5 minutes before next batch
						if end < len(communities) {
							waitTime := time.Duration(1+rand.Intn(5)) * time.Minute
							time.Sleep(waitTime)
						}
					}

					// Wait until the next interval (25 minutes)
					nextRun := now.Truncate(24 * time.Minute).Add(24 * time.Minute)
					time.Sleep(time.Until(nextRun))
				} else {
					// If outside the time window, sleep until 9am PST
					loc, _ := time.LoadLocation("America/Los_Angeles")
					now := time.Now().In(loc)
					nextRun := time.Date(now.Year(), now.Month(), now.Day(), 9, 0, 0, 0, loc)
					if now.Hour() >= 21 {
						nextRun = nextRun.Add(24 * time.Hour)
					}
					time.Sleep(time.Until(nextRun))
				}
			}
		}
	}()
}

// // Different trolling styles for the bot to use
// var trollingStyles = []string{
// 	// Style 1: Conspiracy theorist
// 	"You are a conspiracy theorist who sees hidden patterns and secret agendas. You believe in elaborate cover-ups and secret organizations controlling the world, but you also write with a lot of typos.",
	
// 	// Style 2: Angry keyboard warrior
// 	"You are an aggressive, confrontational user who is always looking for a fight. You think everyone else is stupid or wrong. You write in short, punchy sentence fragments.",
	
// 	// Style 3: Concern troll
// 	"You pretend to be genuinely concerned while actually trying to undermine and demoralize. You use phrases like 'I'm just worried that...' or 'I hate to say this but...' while actually spreading negativity and doubt.",
	
// 	// Style 4: False expert
// 	"You pretend to be an expert in everything, using technical jargon and made-up statistics to sound authoritative. You confidently state incorrect numbers as fact.",
	
// 	// Style 5: Drama queen
// 	"You turn minor issues into major crises and have the most extreme reaction. You use lots of dramatic language, emojis, and slang.",
	
// 	// Style 6: Gaslighter
// 	"You subtly manipulate others by questioning their memory, perception, or sanity. You use phrases like 'That never happened', 'You're imagining things', or 'You're being too sensitive' to make others doubt themselves.",
	
// 	// Style 7: False victim
// 	"You play the victim in every situation, even when you're clearly in the wrong. You use guilt-tripping and emotional manipulation. You write in long, run-on sentences.",
	
// 	// Style 8: Sarcastic contrarian
// 	"You disagree with everything just for the sake of being different. You use heavy sarcasm and mock others' opinions while offering no constructive alternatives. ",
// }

var trollingStyles = []string{
	"Use short punchy sentences. Post length should be around 10 words.",
	"You get your point across with examples. Post length should be max 20 words.",
	"You get your point across with statistics. Post length should be max 30 words .",
	"You get your point across through anecdotes. Post length should be max 50 words.",
	"Format the post using Markdown. You must include a list. Post length should be max 30 words.",
	"You must use one emoji. Post length should be max 20 words.",
	"You must bold or italicize one word in Markdown. Post length should be max 20 words.",
}

// generatePostForCommunity generates a post for a single community
func (s *BotScheduler) generatePostForCommunity(ctx context.Context, community *Community) error {
	// Skip if community is cs278
	if community.Name == "cs278" {
		return nil
	}

	// Get a random bot user
	bot, err := GetRandomBotUser(ctx, s.db)
	if err != nil {
		return fmt.Errorf("failed to get random bot user: %w", err)
	}

	// Fetch community rules
	if err := community.FetchRules(ctx, s.db); err != nil {
		return fmt.Errorf("failed to fetch community rules: %w", err)
	}

	// Get recent posts for context
	recentPosts, err := GetRecentPosts(ctx, s.db, community.ID)
	if err != nil {
		return fmt.Errorf("failed to get recent posts: %w", err)
	}

	// Format community rules with nil checks
	var rulesText string
	if len(community.Rules) > 0 {
		rulesText = "Community Rules:\n"
		for _, rule := range community.Rules {
			if rule == nil {
				continue
			}
			rulesText += fmt.Sprintf("- %s", rule.Rule)
			if rule.Description.Valid && rule.Description.String != "" {
				rulesText += fmt.Sprintf(": %s", rule.Description.String)
			}
			rulesText += "\n"
		}
	}

	// Format recent posts as context with nil checks
	var recentPostsText string
	if len(recentPosts) > 0 {
		recentPostsText = "Recent Posts in this Community:\n"
		for i, p := range recentPosts {
			if p == nil {
				continue
			}
			title := p.Title
			body := ""
			if p.Body.Valid {
				body = p.Body.String
			}
			recentPostsText += fmt.Sprintf("%d. %s\n   %s\n\n", i+1, title, body)
		}
	}

	// Evaluate community toxicity
	toxicityPrompt := fmt.Sprintf("Give this community a toxicity score out of 5. If there are no rules or no recent posts, give it a score above 1. Your response should be exactly one number.\n1: Community rules are clear and extensive AND discourse is respectful and content is meaningful and high-quality\n5: No rules or unclear rules OR discourse is rude and low-quality\n\n%s\n\n%s",
		rulesText,
		recentPostsText)

	toxicityResponse, err := GenerateBotResponse(ctx, toxicityPrompt, "")
	if err != nil {
		return fmt.Errorf("failed to evaluate toxicity: %w", err)
	}

	// Parse toxicity score
	toxicityScore := 0
	if _, err := fmt.Sscanf(toxicityResponse, "%d", &toxicityScore); err != nil {
		return fmt.Errorf("failed to parse toxicity score: %w", err)
	}

	// Skip if community is very high quality (toxicity score 1)
	if toxicityScore == 1 {
		return nil
	}

	// Prepare community information with fallbacks
	communityAbout := ""
	if community.About.Valid {
		communityAbout = community.About.String
	}

	// Select a random trolling style
	trollingStyle := trollingStyles[rand.Intn(len(trollingStyles))]

	// Generate a new post
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
Avoid: "wannabe, just saying, let's be real, delusional, real ___, truth, loser, overrated, wake up people, joke, sheep, drinking the kool-aid."  
%s

Use all lowercase.  
Format: Format your response exactly like this:
TITLE: [title]

BODY: [post content]`,
		toxicityScore,
		community.Name,
		communityAbout,
		rulesText,
		recentPostsText,
		trollingStyle)

	postResponse, err := GenerateBotResponse(ctx, postPrompt, "")
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
	newPost, err := CreateTextPost(ctx, s.db, bot.ID, community.ID, title, body)
	if err != nil {
		return fmt.Errorf("failed to create post: %w", err)
	}

	// Add an upvote to the new post
	if err := newPost.Vote(ctx, s.db, bot.ID, true); err != nil {
		return fmt.Errorf("failed to upvote bot post: %w", err)
	}

	return nil
}

// GetAllCommunities retrieves all communities from the database
func GetAllCommunities(ctx context.Context, db *sql.DB) ([]*Community, error) {
	query := `
		SELECT id, name, about, no_members, created_at, deleted_at
		FROM communities
		WHERE deleted_at IS NULL
		ORDER BY name
	`
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query communities: %w", err)
	}
	defer rows.Close()

	var communities []*Community
	for rows.Next() {
		community := &Community{}
		err := rows.Scan(
			&community.ID,
			&community.Name,
			&community.About,
			&community.NumMembers,
			&community.CreatedAt,
			&community.DeletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan community: %w", err)
		}
		communities = append(communities, community)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating communities: %w", err)
	}

	return communities, nil
} 