package intelligence

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/termiflow/termiflow/internal/providers/llm"
)

// Summarize generates a concise summary of content for a given topic
func Summarize(ctx context.Context, provider llm.Provider, topic, title, content string) (string, error) {
	prompt := fmt.Sprintf(`Summarize the following article in 2-3 sentences for a developer interested in "%s".
Focus on the key technical insights and why it matters.

Title: %s
Content: %s

Summary:`, topic, title, content)

	resp, err := provider.Complete(ctx, llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: "user", Content: prompt},
		},
		MaxTokens:   200,
		Temperature: 0.5,
	})
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(resp.Content), nil
}

// ScoreRelevance scores content relevance to a topic (0.0-1.0)
func ScoreRelevance(ctx context.Context, provider llm.Provider, topic, title, snippet string) (float64, error) {
	prompt := fmt.Sprintf(`You are evaluating if a piece of content is relevant to a user's topic subscription.

Topic: %s
Content Title: %s
Content Snippet: %s

Rate the relevance from 0.0 to 1.0 where:
- 0.0-0.3: Not relevant
- 0.4-0.6: Somewhat relevant
- 0.7-0.9: Highly relevant
- 1.0: Perfectly relevant

Respond with only a number between 0.0 and 1.0.`, topic, title, snippet)

	resp, err := provider.Complete(ctx, llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: "user", Content: prompt},
		},
		MaxTokens:   10,
		Temperature: 0.1,
	})
	if err != nil {
		return 0, err
	}

	score, err := strconv.ParseFloat(strings.TrimSpace(resp.Content), 64)
	if err != nil {
		return 0.5, nil // Default to neutral if parsing fails
	}

	if score < 0 {
		score = 0
	} else if score > 1 {
		score = 1
	}

	return score, nil
}

// ExtractTags extracts relevant tags from content
func ExtractTags(ctx context.Context, provider llm.Provider, title, content string) ([]string, error) {
	prompt := fmt.Sprintf(`Extract 2-4 relevant technical tags from this content. Return only lowercase tags separated by commas.

Title: %s
Content: %s

Tags:`, title, content)

	resp, err := provider.Complete(ctx, llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: "user", Content: prompt},
		},
		MaxTokens:   50,
		Temperature: 0.3,
	})
	if err != nil {
		return nil, err
	}

	tagStr := strings.TrimSpace(resp.Content)
	tags := strings.Split(tagStr, ",")

	var cleanTags []string
	for _, tag := range tags {
		tag = strings.TrimSpace(strings.ToLower(tag))
		tag = strings.TrimPrefix(tag, "#")
		if tag != "" {
			cleanTags = append(cleanTags, tag)
		}
	}

	return cleanTags, nil
}
