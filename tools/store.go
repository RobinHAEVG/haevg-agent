package tools

import (
	"log/slog"
	"net/http"
	"sync"

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
	adClient   *ADClient // azure devops client
	mut        *sync.Mutex
}

// NewStore creates a Store pre-seeded with some sample wiki content.
func NewStore(appConfig *configuration.AppConfig, workDir string, logger *slog.Logger, verbose bool, httpClient *http.Client) *Store {
	s := &Store{
		verbose:    verbose,
		appConfig:  appConfig,
		workDir:    workDir,
		logger:     logger,
		httpClient: httpClient,
		adClient:   NewADClient(httpClient, appConfig.AzureDevops.Organization, appConfig.AzureDevops.Project, appConfig.AzureDevops.APIKey),
		mut:        &sync.Mutex{},
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
