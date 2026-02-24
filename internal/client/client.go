package client

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type Client struct {
	conn    *websocket.Conn
	writeMu sync.Mutex // serializes WriteJSON only
	nextID  atomic.Int64

	pendingMu sync.Mutex
	pending   map[int64]chan rpcResponse

	done    chan struct{} // closed when readLoop exits
	doneErr error        // fatal error from readLoop
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
		conn:    conn,
		pending: make(map[int64]chan rpcResponse),
		done:    make(chan struct{}),
	}

	go c.readLoop()

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
		<-c.done
		return nil, fmt.Errorf("authentication failed: %w", err)
	}
	if !loginResult {
		conn.Close()
		<-c.done
		return nil, fmt.Errorf("authentication failed: login returned false")
	}

	tflog.Debug(ctx, "Successfully connected and authenticated to TrueNAS")

	return c, nil
}

func (c *Client) readLoop() {
	defer close(c.done)

	for {
		var resp rpcResponse
		if err := c.conn.ReadJSON(&resp); err != nil {
			c.pendingMu.Lock()
			c.doneErr = fmt.Errorf("WebSocket read error: %w", err)
			pendingChans := make([]chan rpcResponse, 0, len(c.pending))
			for id, ch := range c.pending {
				pendingChans = append(pendingChans, ch)
				delete(c.pending, id)
			}
			c.pendingMu.Unlock()

			errResp := rpcResponse{Error: &rpcError{
				Code:    -1,
				Message: c.doneErr.Error(),
			}}
			for _, ch := range pendingChans {
				ch <- errResp
			}
			return
		}

		// Skip notifications (no ID)
		if resp.ID == nil {
			continue
		}

		c.pendingMu.Lock()
		ch, ok := c.pending[*resp.ID]
		if ok {
			delete(c.pending, *resp.ID)
		}
		c.pendingMu.Unlock()

		if ok {
			ch <- resp
		}
	}
}

func (c *Client) Call(ctx context.Context, method string, params any, dest any) error {
	// Fail fast if readLoop has already exited
	select {
	case <-c.done:
		return c.doneErr
	default:
	}

	id := c.nextID.Add(1)

	ch := make(chan rpcResponse, 1)
	c.pendingMu.Lock()
	c.pending[id] = ch
	c.pendingMu.Unlock()

	defer func() {
		c.pendingMu.Lock()
		delete(c.pending, id)
		c.pendingMu.Unlock()
	}()

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

	c.writeMu.Lock()
	writeErr := c.conn.WriteJSON(req)
	c.writeMu.Unlock()
	if writeErr != nil {
		return fmt.Errorf("failed to send JSON-RPC request for %s: %w", method, writeErr)
	}

	select {
	case resp := <-ch:
		if resp.Error != nil {
			return resp.Error
		}
		if dest != nil {
			if err := json.Unmarshal(resp.Result, dest); err != nil {
				return fmt.Errorf("failed to unmarshal result for %s: %w", method, err)
			}
		}
		return nil
	case <-ctx.Done():
		return fmt.Errorf("call to %s cancelled: %w", method, ctx.Err())
	case <-c.done:
		return c.doneErr
	}
}

func (c *Client) Close() error {
	err := c.conn.Close()
	<-c.done
	return err
}
