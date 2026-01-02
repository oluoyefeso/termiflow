package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/oluoyefeso/termiflow/internal/config"
	"github.com/oluoyefeso/termiflow/internal/providers/llm"
	"github.com/oluoyefeso/termiflow/internal/providers/search"
	"github.com/oluoyefeso/termiflow/internal/ui"
)

var askSources int
var askNoSearch bool
var askSave bool

var askCmd = &cobra.Command{
	Use:   "ask <question>",
	Short: "Ask a question and get an AI-powered answer with sources",
	Long: `Ask a question and get an AI-powered answer with sources.

Examples:
  termiflow ask "what are the latest advancements in 3nm chip fabrication?"
  termiflow ask "explain rust's borrow checker" --provider local
  termiflow ask "compare TSMC N3 vs Intel 4" --sources 5`,
	Args: cobra.MinimumNArgs(1),
	RunE: runAsk,
}

func init() {
	askCmd.Flags().IntVar(&askSources, "sources", 5, "number of sources to retrieve")
	askCmd.Flags().BoolVar(&askNoSearch, "no-search", false, "answer from LLM knowledge only, don't search")
	askCmd.Flags().BoolVar(&askSave, "save", false, "save this query to history")
}

func runAsk(cmd *cobra.Command, args []string) error {
	question := strings.Join(args, " ")
	cfg := config.Get()

	fmt.Println(ui.Header("termiflow ask"))
	fmt.Println()

	var sources []search.SearchResult
	var err error

	// Search for sources unless --no-search is set
	if !askNoSearch {
		sp := ui.NewSpinner("Searching...")
		sp.Start()

		sources, err = fetchSources(question, askSources)
		if err != nil {
			sp.Error(fmt.Sprintf("Search failed: %v", err))
			// Continue without sources
		} else {
			sp.Stop()
		}
	}

	// Get LLM provider
	providerName := getProvider()
	llmProvider, err := llm.GetProvider(providerName, cfg)
	if err != nil {
		return err
	}

	if !llmProvider.Available() {
		fmt.Fprint(os.Stderr, formatAPIKeyError(providerName))
		return fmt.Errorf("provider not configured")
	}

	// Build prompt with sources
	prompt := buildPrompt(question, sources)

	sp := ui.NewSpinner("Thinking...")
	sp.Start()

	// Stream the response
	ctx := context.Background()
	chunks, err := llmProvider.Stream(ctx, llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: "system", Content: "You are a helpful assistant that provides accurate, well-researched answers. Use the provided sources to inform your response. Be concise but thorough."},
			{Role: "user", Content: prompt},
		},
		MaxTokens:   2048,
		Temperature: 0.7,
		Stream:      true,
	})
	if err != nil {
		sp.Error(fmt.Sprintf("Failed to get response: %v", err))
		return err
	}

	sp.Stop()
	fmt.Println()

	// Stream output
	for chunk := range chunks {
		if chunk.Error != nil {
			return chunk.Error
		}
		fmt.Print(chunk.Content)
	}
	fmt.Println()

	// Print sources
	if len(sources) > 0 {
		fmt.Println()
		fmt.Println(ui.SmallDivider())
		fmt.Println(ui.BoldStyle.Render(" Sources:"))
		for i, src := range sources {
			fmt.Printf("   [%d] %s - %s\n", i+1, ui.MutedStyle.Render(getDomain(src.URL)), src.Title)
		}
	}

	fmt.Println()
	return nil
}

func fetchSources(query string, limit int) ([]search.SearchResult, error) {
	cfg := config.Get()

	if cfg.Search.Tavily.APIKey == "" {
		return nil, fmt.Errorf("Tavily API key not configured")
	}

	tavily := search.NewTavilyProvider(cfg.Search.Tavily.APIKey)
	return tavily.Search(context.Background(), search.SearchRequest{
		Query:      query,
		MaxResults: limit,
		TimeRange:  "week",
	})
}

func buildPrompt(question string, sources []search.SearchResult) string {
	var sb strings.Builder

	if len(sources) > 0 {
		sb.WriteString("Use the following sources to inform your answer:\n\n")
		for i, src := range sources {
			sb.WriteString(fmt.Sprintf("Source %d: %s\n", i+1, src.Title))
			sb.WriteString(fmt.Sprintf("URL: %s\n", src.URL))
			if src.Snippet != "" {
				sb.WriteString(fmt.Sprintf("Content: %s\n", src.Snippet))
			}
			sb.WriteString("\n")
		}
		sb.WriteString("---\n\n")
	}

	sb.WriteString("Question: ")
	sb.WriteString(question)

	return sb.String()
}

func getDomain(url string) string {
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")
	url = strings.TrimPrefix(url, "www.")
	parts := strings.Split(url, "/")
	if len(parts) > 0 {
		return parts[0]
	}
	return url
}

func formatAPIKeyError(provider string) string {
	return fmt.Sprintf(`
 %s API key not configured

   Run one of:
     termiflow config set providers.%s.api_key YOUR_KEY
     export TERMFLOW_%s_API_KEY=YOUR_KEY

   Or run 'termiflow config init' for interactive setup.

`, ui.ErrorStyle.Render("✗"), provider, strings.ToUpper(provider))
}
