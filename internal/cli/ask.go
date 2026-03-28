package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/oluoyefeso/termiflow/internal/config"
	"github.com/oluoyefeso/termiflow/internal/db"
	"github.com/oluoyefeso/termiflow/internal/providers"
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
			var rle *providers.RateLimitError
			if errors.As(err, &rle) {
				sp.Error(rle.Error())
			} else {
				sp.Error(fmt.Sprintf("Search failed: %v", err))
			}
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

	// Build system prompt with user context (subscriptions, unread counts, mode)
	systemPrompt := "You are a helpful assistant that provides accurate, well-researched answers. Use the provided sources to inform your response. Be concise but thorough."
	if userCtx := db.BuildUserContext(); userCtx != "" {
		systemPrompt += "\n\n" + userCtx
	}

	// Stream the response
	ctx := context.Background()
	chunks, err := llmProvider.Stream(ctx, llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: prompt},
		},
		MaxTokens:   2048,
		Temperature: 0.7,
		Stream:      true,
	})
	if err != nil {
		var rle *providers.RateLimitError
		if errors.As(err, &rle) {
			sp.Error(rle.Error())
		} else {
			sp.Error(fmt.Sprintf("Failed to get response: %v", err))
		}
		return err
	}

	sp.Stop()

	// Stream output, capturing content for --save or --json
	var savedContent strings.Builder
	var streamErr error
	for chunk := range chunks {
		if chunk.Error != nil {
			streamErr = chunk.Error
			break
		}
		if !jsonOutput {
			fmt.Print(chunk.Content)
		}
		if askSave || jsonOutput {
			savedContent.WriteString(chunk.Content)
		}
	}

	// JSON output mode
	if jsonOutput {
		askOut := AskOutputJSON{
			Question: question,
			Answer:   savedContent.String(),
			Sources:  buildSourcesJSON(sources),
		}
		if streamErr != nil {
			return ui.WriteJSONError(askOut, streamErr.Error(), version)
		}
		return ui.WriteJSON(askOut, version)
	}

	if streamErr != nil {
		return streamErr
	}

	fmt.Println()
	fmt.Println()

	// Print sources
	if len(sources) > 0 {
		fmt.Println(ui.SmallDivider())
		fmt.Println(ui.BoldStyle.Render(" Sources:"))
		for i, src := range sources {
			fmt.Printf("   [%d] %s - %s\n", i+1, ui.MutedStyle.Render(getDomain(src.URL)), src.Title)
		}
	}

	// Save to file if --save flag is set
	if askSave {
		path, err := saveAskResult(question, savedContent.String(), sources)
		if err != nil {
			fmt.Fprintf(os.Stderr, "\n %s Failed to save: %v\n", ui.ErrorStyle.Render("✗"), err)
		} else {
			fmt.Printf("\n %s %s\n", ui.SuccessStyle.Render("✓"), ui.MutedStyle.Render("Saved to "+path))
		}
	}

	fmt.Println()
	return nil
}

func fetchSources(query string, limit int) ([]search.SearchResult, error) {
	cfg := config.Get()
	provider, err := search.GetSearchProvider(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize search provider: %w", err)
	}
	if !provider.Available() {
		return nil, fmt.Errorf("no search provider configured — run 'termiflow config init'")
	}
	return provider.Search(context.Background(), search.SearchRequest{
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
			fmt.Fprintf(&sb, "Source %d: %s\n", i+1, src.Title)
			fmt.Fprintf(&sb, "URL: %s\n", src.URL)
			if src.Snippet != "" {
				fmt.Fprintf(&sb, "Content: %s\n", src.Snippet)
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

// saveAskResult writes the question, answer, and sources to a markdown file.
func saveAskResult(question, answer string, sources []search.SearchResult) (string, error) {
	saveDir := getSaveDir()
	if err := os.MkdirAll(saveDir, 0755); err != nil {
		return "", fmt.Errorf("create save directory: %w", err)
	}

	ts := time.Now()
	slug := slugify(question)
	filename := fmt.Sprintf("%s-%s.md", ts.Format("20060102-150405"), slug)
	path := filepath.Join(saveDir, filename)

	var sb strings.Builder
	fmt.Fprintf(&sb, "## %s\n\n", question)
	fmt.Fprintf(&sb, "*%s*\n\n", ts.Format("2006-01-02 15:04"))
	sb.WriteString(answer)
	sb.WriteString("\n")

	if len(sources) > 0 {
		sb.WriteString("\n### Sources\n\n")
		for i, src := range sources {
			fmt.Fprintf(&sb, "%d. [%s](%s)\n", i+1, src.Title, src.URL)
		}
	}

	if err := os.WriteFile(path, []byte(sb.String()), 0644); err != nil {
		return "", fmt.Errorf("write file: %w", err)
	}
	return path, nil
}

// getSaveDir returns the directory for saved ask results, respecting XDG_DATA_HOME.
func getSaveDir() string {
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return filepath.Join(xdg, "termiflow", "saved")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".local", "share", "termiflow", "saved")
}

var nonAlphanumeric = regexp.MustCompile(`[^a-z0-9]+`)

// slugify converts a question into a filesystem-safe slug.
func slugify(s string) string {
	s = strings.ToLower(s)
	s = nonAlphanumeric.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if len(s) > 50 {
		s = s[:50]
		s = strings.TrimRight(s, "-")
	}
	return s
}

// JSON output types for ask command.

type AskOutputJSON struct {
	Question string       `json:"question"`
	Answer   string       `json:"answer"`
	Sources  []SourceJSON `json:"sources"`
}

type SourceJSON struct {
	Title  string `json:"title"`
	URL    string `json:"url"`
	Domain string `json:"domain"`
}

func buildSourcesJSON(sources []search.SearchResult) []SourceJSON {
	result := make([]SourceJSON, 0, len(sources))
	for _, src := range sources {
		result = append(result, SourceJSON{
			Title:  src.Title,
			URL:    src.URL,
			Domain: getDomain(src.URL),
		})
	}
	return result
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
