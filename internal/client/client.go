package client

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type AnwinClient struct {
	serverURL  string
	token      string
	httpClient *http.Client
}

type FileEntry struct {
	RelativePath string `json:"relativePath"`
	Content      string `json:"content"`
	Hash         string `json:"hash"`
	Deleted      bool   `json:"deleted"`
}

type registerRequest struct {
	Platform           string `json:"platform"`
	AgentVersion       string `json:"agentVersion"`
	MachineFingerprint string `json:"machineFingerprint"`
	WatchPath          string `json:"watchPath"`
}

type syncRequest struct {
	Files    []FileEntry `json:"files"`
	Initial  bool        `json:"isInitial"`
	SyncedAt int64       `json:"syncedAt"`
}

func New(serverURL string, token string) *AnwinClient {
	return &AnwinClient{
		serverURL: serverURL,
		token:     token,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig:     &tls.Config{MinVersion: tls.VersionTLS12},
				MaxIdleConnsPerHost: 4,
				IdleConnTimeout:     90 * time.Second,
			},
		},
	}
}

func (c *AnwinClient) Ping() error {
	req, err := http.NewRequest("GET", c.serverURL+"/api/agent/ping", nil)
	if err != nil {
		return fmt.Errorf("cannot create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("cannot reach ANWIN server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		return fmt.Errorf("authentication failed — token may be invalid or revoked")
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	return nil
}

func (c *AnwinClient) Register(platform string, agentVersion string, fingerprint string, watchPath string) error {
	body := registerRequest{
		Platform:           platform,
		AgentVersion:       agentVersion,
		MachineFingerprint: fingerprint,
		WatchPath:          watchPath,
	}

	return c.post("/api/agent/register", body)
}

func (c *AnwinClient) Sync(files []FileEntry, isInitial bool) error {
	body := syncRequest{
		Files:    files,
		Initial:  isInitial,
		SyncedAt: time.Now().UnixMilli(),
	}

	return c.post("/api/agent/sync", body)
}

func (c *AnwinClient) SyncWithRetry(files []FileEntry, isInitial bool, maxRetries int) error {
	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(attempt*attempt) * time.Second
			time.Sleep(backoff)
		}
		lastErr = c.Sync(files, isInitial)
		if lastErr == nil {
			return nil
		}
	}
	return lastErr
}

func (c *AnwinClient) post(path string, body interface{}) error {
	jsonData, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("cannot serialize request: %w", err)
	}

	req, err := http.NewRequest("POST", c.serverURL+path, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("cannot create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		return fmt.Errorf("authentication failed — token may be invalid or revoked")
	}

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server error %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

type AgentCommand struct {
	ID             string `json:"id"`
	CommandType    string `json:"commandType"`
	FilePath       string `json:"filePath"`
	Content        string `json:"content"`
	ShellCommand   string `json:"shellCommand"`
	WorkingDir     string `json:"workingDir"`
	TimeoutSeconds int    `json:"timeoutSeconds"`
}

type commandResult struct {
	Status   string `json:"status"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exitCode"`
	Error    string `json:"error"`
}

func (c *AnwinClient) PollCommands() ([]AgentCommand, error) {
	req, err := http.NewRequest("GET", c.serverURL+"/api/agent/commands", nil)
	if err != nil {
		return nil, fmt.Errorf("cannot create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cannot reach server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		return nil, fmt.Errorf("authentication failed")
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("cannot read response: %w", err)
	}

	var commands []AgentCommand
	if err := json.Unmarshal(body, &commands); err != nil {
		return nil, fmt.Errorf("cannot parse commands: %w", err)
	}

	return commands, nil
}

func (c *AnwinClient) ReportCommandResult(commandID string, status string, stdout string, stderr string, exitCode int, errMsg string) error {
	result := commandResult{
		Status:   status,
		Stdout:   stdout,
		Stderr:   stderr,
		ExitCode: exitCode,
		Error:    errMsg,
	}

	return c.post("/api/agent/command-result/"+commandID, result)
}

func SplitBatches(files []FileEntry, batchSize int) [][]FileEntry {
	if len(files) <= batchSize {
		return [][]FileEntry{files}
	}

	var batches [][]FileEntry
	for i := 0; i < len(files); i += batchSize {
		end := i + batchSize
		if end > len(files) {
			end = len(files)
		}
		batches = append(batches, files[i:end])
	}
	return batches
}
