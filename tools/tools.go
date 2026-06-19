package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/RobinHAEVG/haevg-agent/configuration"
	"github.com/RobinHAEVG/haevg-agent/mcp"
)

// Store holds the shared in-memory state for all tools.
type Store struct {
	logger    *log.Logger
	verbose   bool
	appConfig *configuration.AppConfig
	workDir   string
	mut       *sync.Mutex
}

// NewStore creates a Store pre-seeded with some sample wiki content.
func NewStore(appConfig *configuration.AppConfig, workDir string, logger *log.Logger, verbose bool) *Store {
	s := &Store{
		verbose:   verbose,
		appConfig: appConfig,
		workDir:   workDir,
		logger:    logger,
		mut:       &sync.Mutex{},
	}
	return s
}

// ---------------------------------------------------------------------------
// Register
// ---------------------------------------------------------------------------

// RegisterAll registers all tools on the provided MCP server.
func RegisterAll(srv *mcp.Server, store *Store) {
	srv.RegisterTool(readWikiPageTool(), store.readWikiPage)
}

// ---------------------------------------------------------------------------
// Tool: read_wiki_page
// ---------------------------------------------------------------------------

func readWikiPageTool() mcp.Tool {
	schema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"page_id": {
				"type": "string",
				"description": "Wiki page ID, e.g. \"123\"."
			}
		},
		"required": ["page_id"]
	}`)

	return mcp.Tool{
		Name:        "read_wiki_page",
		Description: "Reads the content of an Azure DevOps wiki page by page ID and returns the full Markdown text.",
		InputSchema: schema,
	}
}

type readWikiPageArgs struct {
	PageID string `json:"page_id"`
}

func (s *Store) readWikiPage(raw json.RawMessage) (string, error) {
	var args readWikiPageArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return "", fmt.Errorf("read_wiki_page: invalid arguments: %w", err)
	}
	if args.PageID == "" {
		return "", fmt.Errorf("read_wiki_page: page_id is required")
	}

	if s.verbose {
		s.logger.Printf("read_wiki_page called with page_id: %q\n", args.PageID)
	}

	s.mut.Lock()
	s.wikiPagesRead++
	s.mut.Unlock()

	if s.ado == nil {
		return "", fmt.Errorf("read_wiki_page: Azure DevOps client is not configured")
	}

	content, err := s.ado.ReadWikiPageByID(context.Background(), args.PageID)
	if err != nil {
		return "", fmt.Errorf("read_wiki_page: %w", err)
	}

	return content, nil
}
