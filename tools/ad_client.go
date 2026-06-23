package tools

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

type ADClient struct {
	httpClient   *http.Client
	organization string
	project      string
	apiKey       string
}

func NewADClient(httpClient *http.Client, organization, project, apiKey string) *ADClient {
	return &ADClient{
		httpClient:   httpClient,
		organization: organization,
		project:      project,
		apiKey:       apiKey,
	}
}

func (c *ADClient) GetLatestPipelineLogs(pipelineId, runId string) (string, error) {
	if c == nil {
		return "", fmt.Errorf("azure devops client is nil")
	}
	if c.httpClient == nil {
		return "", fmt.Errorf("azure devops http client is nil")
	}
	if strings.TrimSpace(c.organization) == "" {
		return "", fmt.Errorf("azure devops organization is required")
	}
	if strings.TrimSpace(c.project) == "" {
		return "", fmt.Errorf("azure devops project is required")
	}
	if strings.TrimSpace(pipelineId) == "" {
		return "", fmt.Errorf("pipelineId is required")
	}
	if strings.TrimSpace(runId) == "" {
		return "", fmt.Errorf("runId is required")
	}

	logIndexURL := fmt.Sprintf(
		"https://dev.azure.com/%s/%s/_apis/pipelines/%s/runs/%s/logs?api-version=7.1",
		url.PathEscape(c.organization),
		url.PathEscape(c.project),
		url.PathEscape(pipelineId),
		url.PathEscape(runId),
	)

	req, err := http.NewRequest(http.MethodGet, logIndexURL, nil)
	if err != nil {
		return "", fmt.Errorf("build logs index request: %w", err)
	}
	req.SetBasicAuth("", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request logs index: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))
		return "", fmt.Errorf("logs index request failed: status=%d body=%q", resp.StatusCode, string(body))
	}

	var index pipelineRunLogsResponse
	if err := json.NewDecoder(resp.Body).Decode(&index); err != nil {
		return "", fmt.Errorf("decode logs index response: %w", err)
	}

	if len(index.Value) == 0 {
		return "", fmt.Errorf("no logs available for pipelineId=%s runId=%s", pipelineId, runId)
	}

	latest := index.Value[0]
	for i := 1; i < len(index.Value); i++ {
		if index.Value[i].ID > latest.ID {
			latest = index.Value[i]
		}
	}

	if strings.TrimSpace(latest.URL) == "" {
		return "", fmt.Errorf("latest log entry has empty URL for pipelineId=%s runId=%s", pipelineId, runId)
	}

	logReq, err := http.NewRequest(http.MethodGet, latest.URL, nil)
	if err != nil {
		return "", fmt.Errorf("build latest log request: %w", err)
	}

	if pat := readAzureDevOpsPAT(); pat != "" {
		logReq.SetBasicAuth("", pat)
	}

	logResp, err := c.httpClient.Do(logReq)
	if err != nil {
		return "", fmt.Errorf("request latest log content: %w", err)
	}
	defer logResp.Body.Close()

	if logResp.StatusCode < http.StatusOK || logResp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(logResp.Body, 8192))
		return "", fmt.Errorf("latest log request failed: status=%d body=%q", logResp.StatusCode, string(body))
	}

	body, err := io.ReadAll(logResp.Body)
	if err != nil {
		return "", fmt.Errorf("read latest log content: %w", err)
	}

	return string(body), nil
}

type pipelineRunLogsResponse struct {
	Count int                     `json:"count"`
	Value []pipelineRunLogSummary `json:"value"`
}

type pipelineRunLogSummary struct {
	ID  int    `json:"id"`
	URL string `json:"url"`
}

func readAzureDevOpsPAT() string {
	for _, key := range []string{"AZURE_DEVOPS_PAT", "AZDO_PAT", "AZURE_DEVOPS_EXT_PAT"} {
		if v := strings.TrimSpace(os.Getenv(key)); v != "" {
			return v
		}
	}

	return ""
}
