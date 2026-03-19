// ============================================================
// MCP (Model Context Protocol) - Server Example in One File
// ============================================================
//
// This file shows how to build an MCP server end-to-end:
//
//   1. DATA    - A dataset of anime quotes used by the tool
//   2. TOOL    - An MCP tool that queries the dataset
//   3. SERVER  - A stdio MCP server for use with Claude Desktop
//
// Claude Desktop spawns this process and communicates via stdin/stdout.
//
// claude_desktop_config.json:
//
//   {
//     "mcpServers": {
//       "quote-server": {
//         "command": "go",
//         "args": ["run", "/absolute/path/to/main.go"]
//       }
//     }
//   }
//
// ============================================================

package main

import (
	"context"
	"fmt"
	"log"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ============================================================
// 1. DATA — Quotes from "Classroom of the Elite" anime
// ============================================================

// EpisodeQuote holds a single quote tied to a season and episode.
type EpisodeQuote struct {
	Season  int
	Episode int
	Quote   string
	Author  string
	Source  string
}

// Format returns a human-readable string for the quote.
func (q EpisodeQuote) Format() string {
	return fmt.Sprintf("[S%02dE%02d] \"%s\" — %s (%s)", q.Season, q.Episode, q.Quote, q.Author, q.Source)
}

// quotes is the in-memory dataset that the tool will query.
var quotes = []EpisodeQuote{
	// Season 1
	{1, 1, "What is bad? — Everything that stems from weakness.", "F.W. Nietzsche", "The Antichrist"},
	{1, 2, "It requires great talent to conceal one's talent.", "La Rochefoucauld", "Moral Maxims"},
	{1, 3, "Man is the only animal that makes bargains.", "Adam Smith", "The Wealth of Nations"},
	{1, 4, "We should not be upset that others hide the truth from us, when we so often hide it from ourselves.", "La Rochefoucauld", "Moral Maxims"},
	{1, 5, "Hell is other people.", "Jean-Paul Sartre", "No Exit"},
	{1, 6, "There are two kinds of lies; one concerns a past fact, the other concerns a future duty.", "Jean-Jacques Rousseau", "Emile, or On Education"},
	{1, 7, "Nothing is as dangerous as an ignorant friend; a wise enemy is worth more.", "Jean de La Fontaine", "Fables"},
	{1, 8, "Abandon all hope, ye who enter here.", "Dante Alighieri", "The Divine Comedy (Inferno)"},
	{1, 9, "Man is condemned to be free.", "Jean-Paul Sartre", "Existentialism Is a Humanism"},
	{1, 10, "The most dangerous traitor of all is the one every person carries within.", "Kierkegaard", "Works of Love"},
	{1, 11, "What people commonly call fate is mostly their own stupidity.", "Schopenhauer", "Parerga and Paralipomena"},
	{1, 12, "Genius lives only one floor above madness.", "Schopenhauer", "Parerga and Paralipomena"},

	// Season 2
	{2, 1, "Remember to keep a cool head at all times.", "Horace", "Odes"},
	{2, 2, "There are two main human sins: impatience and indolence.", "Franz Kafka", ""},
	{2, 3, "The greatest souls are capable of the greatest vices as well as of the greatest virtues.", "René Descartes", ""},
	{2, 4, "The material has to be created.", "F. Nightingale", "Notes on Nursing"},
	{2, 5, "Every failure is a step toward success.", "W. Whewell", "Lectures on the History of Moral Philosophy"},
	{2, 6, "Adversity is the first path to truth.", "G.G. Byron", ""},
	{2, 7, "Doubt everything or believe everything — both save thinking.", "H. Poincaré", "Science and Hypothesis"},
	{2, 8, "The wound lives silently within the heart.", "Virgil", "Aeneid"},
	{2, 9, "If you make a mistake and do not correct it, that is called a mistake.", "Anonymous", "Analects"},
	{2, 10, "People often desire their own ruin, deceived by a false appearance of good.", "N. Machiavelli", "Discourses on Livy"},
	{2, 11, "He who cannot command himself will always be a slave.", "J.W.V. Goethe", "Zahme Xenien"},
	{2, 12, "The strength of a plan is exhausted by its own mass.", "Horace", "Odes"},
	{2, 13, "The worst enemy you will ever meet will always be yourself.", "F.W. Nietzsche", "Thus Spoke Zarathustra"},

	// Season 3
	{3, 1, "The strongest principle of growth lies in human choice.", "George Eliot", "Daniel Deronda"},
	{3, 2, "Man is a wolf to man.", "Plautus", "Asinaria"},
	{3, 3, "We do not forget what we strive to forget.", "F.W. Nietzsche", "Thus Spoke Zarathustra"},
	{3, 4, "You have the right to work, but not to the fruits of your work.", "Unknown", "Bhagavad Gita"},
	{3, 5, "Fortune favors the bold.", "Virgil", "Aeneid"},
	{3, 6, "It is better to suffer injustice than to commit it.", "Cicero", "Tusculan Disputations"},
	{3, 7, "People will do anything, no matter how absurd, to avoid confronting their own souls.", "Carl Jung", "Psychology and Alchemy"},
	{3, 8, "Those who cannot remember the past are condemned to repeat it.", "George Santayana", "The Life of Reason"},
	{3, 9, "Extreme justice leads to extreme injustice.", "Cicero", "De Officiis"},
	{3, 10, "The first cause of absurd conclusions I attribute to lack of method.", "Thomas Hobbes", "Leviathan"},
	{3, 11, "There is only one law in emotion: to bring happiness to those we love.", "Stendhal", "Diary"},
	{3, 12, "Do not change the world to fit your desires; first change yourself.", "Descartes", "Discourse on the Method"},
	{3, 13, "Love is the best teacher.", "Pliny the Younger", "Epistulae"},
}

// getQuote searches the dataset by season and episode number.
func getQuote(season, episode int) (*EpisodeQuote, error) {
	for _, q := range quotes {
		if q.Season == season && q.Episode == episode {
			return &q, nil
		}
	}
	return nil, fmt.Errorf("no quote found for season %d, episode %d", season, episode)
}

// ============================================================
// 2. TOOL — MCP tool definition and handler
// ============================================================

// QuoteInput is the structured input the LLM must provide when calling the tool.
type QuoteInput struct {
	Season  int `json:"season"`
	Episode int `json:"episode"`
}

// QuoteOutput is the structured result the tool returns to the LLM.
type QuoteOutput struct {
	Quote string `json:"quote"`
}

// quoteToolHandler is the function registered as an MCP tool.
// The go-sdk automatically decodes the JSON arguments into QuoteInput.
func quoteToolHandler(
	_ context.Context,
	req *mcp.CallToolRequest,
	input QuoteInput,
) (*mcp.CallToolResult, QuoteOutput, error) {
	q, err := getQuote(input.Season, input.Episode)
	if err != nil {
		return nil, QuoteOutput{}, err
	}
	return nil, QuoteOutput{Quote: q.Format()}, nil
}

// ============================================================
// 3. SERVER — Stdio MCP server (used by Claude Desktop)
// ============================================================

func main() {
	// NewServer creates an MCP server instance with basic metadata.
	server := mcp.NewServer(
		&mcp.Implementation{Name: "quote-server", Version: "1.0.0"},
		nil,
	)

	// Register the tool: provide a Tool descriptor and the handler function.
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_quote",
		Description: "Returns an inspirational quote from Classroom of the Elite for the given season and episode.",
	}, quoteToolHandler)

	// StdioTransport communicates via stdin/stdout.
	// Claude Desktop spawns this process and talks to it directly — no HTTP needed.
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatal(err)
	}
}
