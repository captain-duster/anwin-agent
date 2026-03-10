package client

import (
	"bytes"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/anwin/agent/internal/logger"
)

const (
	maxRetries      = 3
	initialBackoff  = 1 * time.Second
	requestTimeout  = 30 * time.Second
	maxResponseBody = 1024 * 1024
)

type Client struct {
	serverURL    string
	token        string
	agentVersion string
	httpClient   *http.Client
}

type RegisterRequest struct {
	ProjectID          string `json:"projectId"`
	Platform           string `json:"platform"`
	AgentVersion       string `json:"agentVersion"`
	MachineFingerprint string `json:"machineFingerprint"`
	WatchPath          string `json:"watchPath"`
}

type SyncRequest struct {
	ProjectID string      `json:"projectId"`
	Files     []FileEntry `json:"files"`
	IsInitial bool        `json:"isInitial"`
	SyncedAt  int64       `json:"syncedAt"`
}

type FileEntry struct {
	RelativePath string `json:"relativePath"`
	Content      string `json:"content"`
	Hash         string `json:"hash"`
	Deleted      bool   `json:"deleted"`
}

type apiError struct {
	statusCode int
	permanent  bool
	message    string
}

func (e *apiError) Error() string {
	return e.message
}

func New(serverURL, token, agentVersion string) *Client {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
			CipherSuites: []uint16{
				tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
				tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			},
		},
		DisableKeepAlives:     false,
		MaxIdleConns:          10,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	return &Client{
		serverURL:    serverURL,
		token:        token,
		agentVersion: agentVersion,
		httpClient: &http.Client{
			Timeout:   requestTimeout,
			Transport: transport,
		},
	}
}

func (c *Client) post(endpoint string, body interface{}) error {
	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("serialization failed: %w", err)
	}

	var lastErr error
	backoff := initialBackoff

	for attempt := 1; attempt <= maxRetries; attempt++ {
		req, err := http.NewRequest("POST", c.serverURL+endpoint, bytes.NewBuffer(data))
		if err != nil {
			return fmt.Errorf("failed to build request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+c.token)
		req.Header.Set("X-Agent-Version", c.agentVersion)
		req.Header.Set("X-Agent-Platform", Platform())
		req.Header.Set("X-Machine-Fingerprint", MachineFingerprint())
		req.Header.Set("User-Agent", "ANWIN-Agent/"+c.agentVersion)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			if attempt < maxRetries {
				logger.Warn("Request failed, retrying",
					"attempt", fmt.Sprintf("%d/%d", attempt, maxRetries),
					"error", err.Error(),
					"retrying_in", backoff.String(),
				)
				time.Sleep(backoff)
				backoff *= 2
			}
			continue
		}

		io.LimitReader(resp.Body, maxResponseBody)
		resp.Body.Close()

		apiErr := toAPIError(resp.StatusCode)
		if apiErr != nil {
			if apiErr.permanent {
				return apiErr
			}
			lastErr = apiErr
			if attempt < maxRetries {
				logger.Warn("Server error, retrying",
					"attempt", fmt.Sprintf("%d/%d", attempt, maxRetries),
					"status", resp.StatusCode,
					"retrying_in", backoff.String(),
				)
				time.Sleep(backoff)
				backoff *= 2
			}
			continue
		}

		return nil
	}

	return fmt.Errorf("all %d attempts failed: %w", maxRetries, lastErr)
}

func (c *Client) RegisterAgent(projectID, watchPath string) error {
	return c.post("/api/agent/register", RegisterRequest{
		ProjectID:          projectID,
		Platform:           Platform(),
		AgentVersion:       c.agentVersion,
		MachineFingerprint: MachineFingerprint(),
		WatchPath:          watchPath,
	})
}

func (c *Client) SyncFiles(projectID string, files []FileEntry, isInitial bool) error {
	return c.post("/api/agent/sync", SyncRequest{
		ProjectID: projectID,
		Files:     files,
		IsInitial: isInitial,
		SyncedAt:  time.Now().Unix(),
	})
}

func (c *Client) Ping() bool {
	req, err := http.NewRequest("GET", c.serverURL+"/api/agent/ping", nil)
	if err != nil {
		return false
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("X-Agent-Version", c.agentVersion)
	req.Header.Set("User-Agent", "ANWIN-Agent/"+c.agentVersion)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200
}

func toAPIError(statusCode int) *apiError {
	switch statusCode {
	case 200, 201, 204:
		return nil
	case 401:
		return &apiError{statusCode: 401, permanent: true, message: "authentication failed — check your agent token"}
	case 403:
		return &apiError{statusCode: 403, permanent: true, message: "access denied — token may have been revoked"}
	case 404:
		return &apiError{statusCode: 404, permanent: true, message: "project not found — check your project ID"}
	case 429:
		return &apiError{statusCode: 429, permanent: false, message: "rate limited by server"}
	default:
		return &apiError{statusCode: statusCode, permanent: false, message: fmt.Sprintf("server error: %d", statusCode)}
	}
}

func Platform() string {
	switch runtime.GOOS {
	case "windows":
		return "WINDOWS"
	case "darwin":
		return "MAC"
	default:
		return "LINUX"
	}
}

func MachineFingerprint() string {
	hostname, _ := os.Hostname()
	var seed string
	switch runtime.GOOS {
	case "windows":
		seed = hostname + os.Getenv("USERNAME") + os.Getenv("COMPUTERNAME")
	default:
		seed = hostname + os.Getenv("USER") + runtime.GOOS
	}
	hash := sha256.Sum256([]byte(seed))
	return hex.EncodeToString(hash[:])
}
