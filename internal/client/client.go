package client

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type Client struct {
	conn   *websocket.Conn
	mu     sync.Mutex
	nextID int64
}

type rpcRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int64  `json:"id"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *int64          `json:"id"`
	Result  json.RawMessage `json:"result"`
	Error   *rpcError       `json:"error"`
}

type rpcError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

func (e *rpcError) Error() string {
	if len(e.Data) > 0 {
		return fmt.Sprintf("JSON-RPC error %d: %s (data: %s)", e.Code, e.Message, string(e.Data))
	}
	return fmt.Sprintf("JSON-RPC error %d: %s", e.Code, e.Message)
}

func NewClient(ctx context.Context, wsURL, apiKey string, insecure bool) (*Client, error) {
	url := strings.TrimRight(wsURL, "/") + "/api/current"

	tflog.Debug(ctx, "Connecting to TrueNAS WebSocket", map[string]any{"url": url})

	dialer := websocket.Dialer{}
	if insecure {
		dialer.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	}

	header := http.Header{}
	conn, _, err := dialer.DialContext(ctx, url, header)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to TrueNAS WebSocket at %s: %w", url, err)
	}

	c := &Client{
		conn:   conn,
		nextID: 1,
	}

	// Authenticate with API key (retry on rate limit)
	const maxRetries = 5
	backoff := 5 * time.Second
	var loginResult bool
	for attempt := range maxRetries {
		loginResult = false
		err = c.Call(ctx, "auth.login_with_api_key", []string{apiKey}, &loginResult)
		if err != nil && strings.Contains(err.Error(), "Rate Limit") && attempt < maxRetries-1 {
			tflog.Warn(ctx, "Rate limited during authentication, retrying", map[string]any{
				"attempt": attempt + 1,
				"backoff": backoff.String(),
			})
			time.Sleep(backoff)
			backoff *= 2
			continue
		}
		break
	}
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("authentication failed: %w", err)
	}
	if !loginResult {
		conn.Close()
		return nil, fmt.Errorf("authentication failed: login returned false")
	}

	tflog.Debug(ctx, "Successfully connected and authenticated to TrueNAS")

	return c, nil
}

func (c *Client) Call(ctx context.Context, method string, params any, dest any) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	id := c.nextID
	c.nextID++

	req := rpcRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}

	tflog.Debug(ctx, "Sending JSON-RPC request", map[string]any{
		"method": method,
		"id":     id,
	})

	if err := c.conn.WriteJSON(req); err != nil {
		return fmt.Errorf("failed to send JSON-RPC request for %s: %w", method, err)
	}

	for {
		var resp rpcResponse
		if err := c.conn.ReadJSON(&resp); err != nil {
			return fmt.Errorf("failed to read JSON-RPC response for %s: %w", method, err)
		}

		// Skip notification messages (no id field)
		if resp.ID == nil {
			tflog.Trace(ctx, "Skipping WebSocket notification", map[string]any{
				"method": method,
			})
			continue
		}

		// Skip responses with non-matching IDs
		if *resp.ID != id {
			tflog.Trace(ctx, "Skipping response with non-matching ID", map[string]any{
				"expected_id": id,
				"received_id": *resp.ID,
			})
			continue
		}

		if resp.Error != nil {
			return resp.Error
		}

		if dest != nil {
			if err := json.Unmarshal(resp.Result, dest); err != nil {
				return fmt.Errorf("failed to unmarshal result for %s: %w", method, err)
			}
		}

		return nil
	}
}

// CallJob calls a method that returns a job ID and waits for the job to complete.
// The job result is unmarshaled into dest. It uses core.job_wait which blocks
// until the job finishes and returns the result directly.
func (c *Client) CallJob(ctx context.Context, method string, params any, dest any) error {
	var jobID int64
	if err := c.Call(ctx, method, params, &jobID); err != nil {
		return err
	}

	tflog.Debug(ctx, "Waiting for job to complete", map[string]any{
		"method": method,
		"job_id": jobID,
	})

	return c.Call(ctx, "core.job_wait", []any{jobID}, dest)
}

func (c *Client) Close() error {
	return c.conn.Close()
}
