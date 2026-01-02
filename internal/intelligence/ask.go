package intelligence

import (
	"context"

	"github.com/oluoyefeso/termiflow/internal/providers/llm"
	"github.com/oluoyefeso/termiflow/internal/providers/search"
)

type AskResult struct {
	Answer  string
	Sources []search.SearchResult
}

// Ask performs a search and generates an answer using the LLM
func Ask(ctx context.Context, question string, llmProvider llm.Provider, searchProvider search.Provider, maxSources int) (*AskResult, error) {
	// Search for relevant sources
	var sources []search.SearchResult
	if searchProvider != nil && searchProvider.Available() {
		results, err := searchProvider.Search(ctx, search.SearchRequest{
			Query:      question,
			MaxResults: maxSources,
			TimeRange:  "week",
		})
		if err == nil {
			sources = results
		}
	}

	// Build prompt with sources
	prompt := buildPromptWithSources(question, sources)

	// Get LLM response
	resp, err := llmProvider.Complete(ctx, llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: "system", Content: "You are a helpful assistant that provides accurate, well-researched answers. Use the provided sources to inform your response. Be concise but thorough."},
			{Role: "user", Content: prompt},
		},
		MaxTokens:   2048,
		Temperature: 0.7,
	})
	if err != nil {
		return nil, err
	}

	return &AskResult{
		Answer:  resp.Content,
		Sources: sources,
	}, nil
}

func buildPromptWithSources(question string, sources []search.SearchResult) string {
	prompt := ""

	if len(sources) > 0 {
		prompt += "Use the following sources to inform your answer:\n\n"
		for i, src := range sources {
			prompt += "Source " + string(rune('1'+i)) + ": " + src.Title + "\n"
			prompt += "URL: " + src.URL + "\n"
			if src.Snippet != "" {
				prompt += "Content: " + src.Snippet + "\n"
			}
			prompt += "\n"
		}
		prompt += "---\n\n"
	}

	prompt += "Question: " + question
	return prompt
}
