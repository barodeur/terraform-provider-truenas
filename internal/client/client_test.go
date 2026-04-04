package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestMain(m *testing.M) {
	jobPollInterval = 50 * time.Millisecond
	os.Exit(m.Run())
}

// mockServer is a WebSocket server that handles JSON-RPC requests
// with configurable method handlers.
type mockServer struct {
	server   *httptest.Server
	handlers map[string]func(id int64, params json.RawMessage) (any, *rpcError)
	mu       sync.Mutex
}

func newMockServer(t *testing.T) *mockServer {
	t.Helper()

	ms := &mockServer{
		handlers: make(map[string]func(id int64, params json.RawMessage) (any, *rpcError)),
	}

	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

	ms.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade failed: %v", err)
			return
		}
		defer conn.Close()

		for {
			var req rpcRequest
			if err := conn.ReadJSON(&req); err != nil {
				return
			}

			ms.mu.Lock()
			handler, ok := ms.handlers[req.Method]
			ms.mu.Unlock()

			var resp rpcResponse
			resp.JSONRPC = "2.0"
			resp.ID = &req.ID

			if !ok {
				resp.Error = &rpcError{Code: -32601, Message: "method not found: " + req.Method}
			} else {
				raw, _ := json.Marshal(req.Params)
				result, rpcErr := handler(req.ID, raw)
				if rpcErr != nil {
					resp.Error = rpcErr
				} else {
					resultBytes, _ := json.Marshal(result)
					resp.Result = resultBytes
				}
			}

			if err := conn.WriteJSON(resp); err != nil {
				return
			}
		}
	}))

	return ms
}

func (ms *mockServer) handle(method string, fn func(id int64, params json.RawMessage) (any, *rpcError)) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.handlers[method] = fn
}

func (ms *mockServer) wsURL() string {
	return "ws" + strings.TrimPrefix(ms.server.URL, "http")
}

func (ms *mockServer) close() {
	ms.server.Close()
}

// connectMock creates a Client connected to the mock server.
// It sets up auth.login_with_api_key to succeed.
func connectMock(t *testing.T, ms *mockServer) *Client {
	t.Helper()

	ms.handle("auth.login_with_api_key", func(_ int64, _ json.RawMessage) (any, *rpcError) {
		return true, nil
	})

	ctx := context.Background()
	// The client appends /api/current, but the mock server doesn't care about the path.
	// We need the URL without the path suffix since NewClient adds it.
	c, err := NewClient(ctx, ms.wsURL(), "test-api-key", false)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	t.Cleanup(func() { c.Close() })
	return c
}

func TestCall_Success(t *testing.T) {
	ms := newMockServer(t)
	defer ms.close()

	ms.handle("test.method", func(_ int64, _ json.RawMessage) (any, *rpcError) {
		return map[string]string{"hello": "world"}, nil
	})

	c := connectMock(t, ms)

	var result map[string]string
	err := c.Call(context.Background(), "test.method", nil, &result)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}
	if result["hello"] != "world" {
		t.Fatalf("unexpected result: %v", result)
	}
}

func TestCall_RPCError(t *testing.T) {
	ms := newMockServer(t)
	defer ms.close()

	ms.handle("test.fail", func(_ int64, _ json.RawMessage) (any, *rpcError) {
		return nil, &rpcError{Code: 42, Message: "something broke"}
	})

	c := connectMock(t, ms)

	err := c.Call(context.Background(), "test.fail", nil, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "something broke") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCall_ContextCancelled(t *testing.T) {
	ms := newMockServer(t)
	defer ms.close()

	// Handler that never responds (blocks forever)
	ms.handle("test.slow", func(_ int64, _ json.RawMessage) (any, *rpcError) {
		time.Sleep(10 * time.Second)
		return nil, nil
	})

	c := connectMock(t, ms)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := c.Call(ctx, "test.slow", nil, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "cancelled") {
		t.Fatalf("expected context cancellation error, got: %v", err)
	}
}

// --- CallJob tests ---

// mockJobState tracks a fake TrueNAS job that transitions through states.
type mockJobState struct {
	mu     sync.Mutex
	state  string
	result any
	err    string
}

// setupJobHandlers wires up the mock server to simulate TrueNAS job behavior:
// - The target method returns a job ID
// - core.get_jobs returns the job's current state
func setupJobHandlers(ms *mockServer, jobID int64, job *mockJobState) {
	// The method that returns a job ID
	ms.handle("pool.create", func(_ int64, _ json.RawMessage) (any, *rpcError) {
		return jobID, nil
	})

	// core.get_jobs polling endpoint
	ms.handle("core.get_jobs", func(_ int64, params json.RawMessage) (any, *rpcError) {
		job.mu.Lock()
		defer job.mu.Unlock()

		return []map[string]any{
			{
				"id":     jobID,
				"state":  job.state,
				"result": job.result,
				"error":  job.err,
				"progress": map[string]any{
					"percent":     100,
					"description": "done",
				},
			},
		}, nil
	})
}

func TestCallJob_Success(t *testing.T) {
	ms := newMockServer(t)
	defer ms.close()

	job := &mockJobState{state: "RUNNING"}
	setupJobHandlers(ms, 42, job)

	c := connectMock(t, ms)

	// Transition job to SUCCESS after a short delay
	go func() {
		time.Sleep(100 * time.Millisecond)
		job.mu.Lock()
		job.state = "SUCCESS"
		job.result = map[string]any{"id": float64(1), "name": "tank"}
		job.mu.Unlock()
	}()

	var result map[string]any
	err := c.CallJob(context.Background(), "pool.create", []any{map[string]string{"name": "tank"}}, &result)
	if err != nil {
		t.Fatalf("CallJob failed: %v", err)
	}
	if result["name"] != "tank" {
		t.Fatalf("unexpected result: %v", result)
	}
}

func TestCallJob_Failed(t *testing.T) {
	ms := newMockServer(t)
	defer ms.close()

	job := &mockJobState{state: "RUNNING"}
	setupJobHandlers(ms, 42, job)

	c := connectMock(t, ms)

	// Transition job to FAILED after a short delay
	go func() {
		time.Sleep(100 * time.Millisecond)
		job.mu.Lock()
		job.state = "FAILED"
		job.err = "disk not found"
		job.mu.Unlock()
	}()

	err := c.CallJob(context.Background(), "pool.create", []any{map[string]string{"name": "tank"}}, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "disk not found") {
		t.Fatalf("expected job error message, got: %v", err)
	}
}

func TestCallJob_Aborted(t *testing.T) {
	ms := newMockServer(t)
	defer ms.close()

	job := &mockJobState{state: "RUNNING"}
	setupJobHandlers(ms, 42, job)

	c := connectMock(t, ms)

	go func() {
		time.Sleep(100 * time.Millisecond)
		job.mu.Lock()
		job.state = "ABORTED"
		job.mu.Unlock()
	}()

	err := c.CallJob(context.Background(), "pool.create", []any{map[string]string{"name": "tank"}}, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "aborted") && !strings.Contains(err.Error(), "ABORTED") {
		t.Fatalf("expected aborted error, got: %v", err)
	}
}

func TestCallJob_ContextTimeout(t *testing.T) {
	ms := newMockServer(t)
	defer ms.close()

	// Job stays in RUNNING forever
	job := &mockJobState{state: "RUNNING"}
	setupJobHandlers(ms, 42, job)

	c := connectMock(t, ms)

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	err := c.CallJob(ctx, "pool.create", []any{map[string]string{"name": "tank"}}, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "cancelled") && !strings.Contains(err.Error(), "context") {
		t.Fatalf("expected context error, got: %v", err)
	}
}

func TestCallJob_ImmediateSuccess(t *testing.T) {
	ms := newMockServer(t)
	defer ms.close()

	// Job is already SUCCESS on first poll
	job := &mockJobState{
		state:  "SUCCESS",
		result: map[string]any{"id": float64(1)},
	}
	setupJobHandlers(ms, 42, job)

	c := connectMock(t, ms)

	var result map[string]any
	err := c.CallJob(context.Background(), "pool.create", []any{map[string]string{"name": "tank"}}, &result)
	if err != nil {
		t.Fatalf("CallJob failed: %v", err)
	}
	if result["id"] != float64(1) {
		t.Fatalf("unexpected result: %v", result)
	}
}

func TestCallJob_WaitingThenSuccess(t *testing.T) {
	ms := newMockServer(t)
	defer ms.close()

	job := &mockJobState{state: "WAITING"}
	setupJobHandlers(ms, 42, job)

	c := connectMock(t, ms)

	go func() {
		time.Sleep(50 * time.Millisecond)
		job.mu.Lock()
		job.state = "RUNNING"
		job.mu.Unlock()

		time.Sleep(50 * time.Millisecond)
		job.mu.Lock()
		job.state = "SUCCESS"
		job.result = "ok"
		job.mu.Unlock()
	}()

	var result string
	err := c.CallJob(context.Background(), "pool.create", []any{map[string]string{"name": "tank"}}, &result)
	if err != nil {
		t.Fatalf("CallJob failed: %v", err)
	}
	if result != "ok" {
		t.Fatalf("unexpected result: %v", result)
	}
}

func TestCallJob_RPCError(t *testing.T) {
	ms := newMockServer(t)
	defer ms.close()

	// The initial method call fails (not a job error, an RPC error)
	ms.handle("pool.create", func(_ int64, _ json.RawMessage) (any, *rpcError) {
		return nil, &rpcError{Code: -1, Message: "permission denied"}
	})

	c := connectMock(t, ms)

	err := c.CallJob(context.Background(), "pool.create", nil, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "permission denied") {
		t.Fatalf("expected RPC error, got: %v", err)
	}
}
