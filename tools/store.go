package tools

import (
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/RobinHAEVG/haevg-agent/configuration"
	"github.com/RobinHAEVG/haevg-agent/mcp"
)

// Store holds the shared in-memory state for all tools.
type Store struct {
	logger     *slog.Logger
	verbose    bool
	appConfig  *configuration.AppConfig
	workDir    string
	httpClient *http.Client
	adClient   *adClient // azure devops client
	mut        *sync.Mutex
}

// NewStore creates a Store pre-seeded with some sample wiki content.
func NewStore(appConfig *configuration.AppConfig, workDir string, logger *slog.Logger, verbose bool, httpClient *http.Client) *Store {
	s := &Store{
		verbose:   verbose,
		appConfig: appConfig,
		workDir:   workDir,
		logger:    logger,
		mut:       &sync.Mutex{},
	}

	if httpClient != nil {
		s.httpClient = httpClient
	} else {
		s.httpClient = &http.Client{
			Timeout: 30 * time.Second,
		}
	}

	return s
}

// RegisterAll registers all tools on the provided MCP server.
func RegisterAll(srv *mcp.Server, store *Store) {
	srv.RegisterTool(readFileTool(), store.readFile)
	srv.RegisterTool(writeFileTool(), store.writeFile)
	srv.RegisterTool(readDirectoryTool(), store.readDirectory)
	srv.RegisterTool(getPackageDocumentationTool(), store.getPackageDocumentation)
	srv.RegisterTool(getLatestPipelineLogsTool(), store.getLatestPipelineLogs)
}
