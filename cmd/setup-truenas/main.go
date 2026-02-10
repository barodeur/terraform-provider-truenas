package main

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/websocket"
)

type rpcRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      *int64 `json:"id,omitempty"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *int64          `json:"id"`
	Method  string          `json:"method,omitempty"`
	Result  json.RawMessage `json:"result"`
	Error   *rpcError       `json:"error"`
	Params  json.RawMessage `json:"params,omitempty"`
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

var nextID int64 = 1

// call sends a JSON-RPC request and reads messages until the matching response arrives.
// Messages without an id are treated as notifications and logged.
func call(conn *websocket.Conn, method string, params any, dest any) error {
	id := nextID
	nextID++

	req := rpcRequest{
		JSONRPC: "2.0",
		ID:      &id,
		Method:  method,
		Params:  params,
	}

	log.Printf("-> %s (id=%d)", method, id)
	if err := conn.WriteJSON(req); err != nil {
		return fmt.Errorf("failed to send %s: %w", method, err)
	}

	for {
		var resp rpcResponse
		if err := conn.ReadJSON(&resp); err != nil {
			return fmt.Errorf("failed to read response for %s: %w", method, err)
		}

		// Notification (no id) — log and continue
		if resp.ID == nil {
			if resp.Method != "" {
				log.Printf("   notification: %s", resp.Method)
			} else {
				log.Printf("   notification: %s", string(resp.Params))
			}
			continue
		}

		// Response with matching id
		if *resp.ID == id {
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

		log.Printf("   unexpected response id=%d (waiting for %d), skipping", *resp.ID, id)
	}
}

func waitForPort(host string, port int, timeout time.Duration) error {
	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	deadline := time.Now().Add(timeout)

	log.Printf("Waiting for %s to be reachable...", addr)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
		if err == nil {
			conn.Close()
			log.Printf("Port %d is reachable", port)
			return nil
		}
		time.Sleep(3 * time.Second)
	}
	return fmt.Errorf("timed out waiting for %s after %s", addr, timeout)
}

func waitForWebSocket(url string, dialer *websocket.Dialer, timeout time.Duration) (*websocket.Conn, error) {
	deadline := time.Now().Add(timeout)
	log.Printf("Waiting for WebSocket at %s...", url)

	for time.Now().Before(deadline) {
		conn, _, err := dialer.Dial(url, http.Header{})
		if err == nil {
			log.Printf("WebSocket connected: %s", url)
			return conn, nil
		}
		time.Sleep(3 * time.Second)
	}
	return nil, fmt.Errorf("timed out waiting for WebSocket at %s after %s", url, timeout)
}

func main() {
	host := flag.String("host", "127.0.0.1", "TrueNAS host")
	port := flag.Int("port", 8080, "TrueNAS HTTP port (installer)")
	httpsPort := flag.Int("https-port", 8443, "TrueNAS HTTPS port (API, used for bootstrap)")
	adminPassword := flag.String("admin-password", "testing123", "Admin password to set during install")
	apiKeyName := flag.String("api-key-name", "terraform-integration-test", "Name of the API key to create")
	poolName := flag.String("pool-name", "tank", "Name of the data pool to create")
	outputFile := flag.String("output-file", "/tmp/truenas-api-key", "File to write the API key to")
	installTimeout := flag.Duration("install-timeout", 15*time.Minute, "Timeout for installation phase")
	bootTimeout := flag.Duration("boot-timeout", 10*time.Minute, "Timeout for post-install boot")
	flag.Parse()

	log.SetFlags(log.Ltime)

	wsDialer := websocket.DefaultDialer
	wssDialer := &websocket.Dialer{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	// Phase 1: Installer
	log.Println("=== Phase 1: TrueNAS Installer ===")

	if err := waitForPort(*host, *port, *installTimeout); err != nil {
		log.Fatalf("Installer not reachable: %v", err)
	}

	installerURL := fmt.Sprintf("ws://%s:%d/ws", *host, *port)
	conn, err := waitForWebSocket(installerURL, wsDialer, 2*time.Minute)
	if err != nil {
		log.Fatalf("Failed to connect to installer: %v", err)
	}
	defer conn.Close()

	// Check adoption status
	var isAdopted bool
	if err := call(conn, "is_adopted", nil, &isAdopted); err != nil {
		log.Fatalf("is_adopted failed: %v", err)
	}
	log.Printf("is_adopted = %v", isAdopted)
	if isAdopted {
		log.Fatal("System is already adopted; cannot run installer")
	}

	// Adopt
	var authKey string
	if err := call(conn, "adopt", nil, &authKey); err != nil {
		log.Fatalf("adopt failed: %v", err)
	}
	log.Printf("Adopted, got auth key")

	// Authenticate
	var authResult bool
	if err := call(conn, "authenticate", []any{authKey}, &authResult); err != nil {
		log.Fatalf("authenticate failed: %v", err)
	}
	if !authResult {
		log.Fatal("authenticate returned false")
	}
	log.Println("Authenticated with installer")

	// List disks
	var disks []map[string]any
	if err := call(conn, "list_disks", nil, &disks); err != nil {
		log.Fatalf("list_disks failed: %v", err)
	}
	if len(disks) == 0 {
		log.Fatal("No disks found")
	}
	diskName, ok := disks[0]["name"].(string)
	if !ok {
		log.Fatalf("Could not get disk name from: %v", disks[0])
	}
	log.Printf("Using disk: %s", diskName)

	// Install
	log.Println("Starting installation (this takes several minutes)...")
	installParams := map[string]any{
		"disks":    []string{diskName},
		"set_pmbr": false,
		"authentication": map[string]any{
			"username": "truenas_admin",
			"password": *adminPassword,
		},
	}

	// Set a long read deadline for the install call
	conn.SetReadDeadline(time.Now().Add(*installTimeout))
	if err := call(conn, "install", []any{installParams}, nil); err != nil {
		log.Fatalf("install failed: %v", err)
	}
	conn.SetReadDeadline(time.Time{})
	log.Println("Installation complete")

	// Reboot
	log.Println("Rebooting...")
	// reboot may not return a clean response, so tolerate errors
	_ = call(conn, "reboot", nil, nil)
	conn.Close()

	// Phase 2: Bootstrap API key (over HTTPS to avoid insecure-transport revocation)
	log.Println("=== Phase 2: Bootstrap API Key ===")

	// Wait a bit for the reboot to start
	log.Println("Waiting for reboot...")
	time.Sleep(15 * time.Second)

	// Connect over WSS (HTTPS) to avoid API key being revoked for insecure transport
	apiURL := fmt.Sprintf("wss://%s:%d/api/current", *host, *httpsPort)

	apiConn, err := waitForWebSocket(apiURL, wssDialer, *bootTimeout)
	if err != nil {
		log.Fatalf("Failed to connect to TrueNAS API after reboot: %v", err)
	}
	defer apiConn.Close()

	// Login with username/password
	var loginResult bool
	if err := call(apiConn, "auth.login", []any{"truenas_admin", *adminPassword}, &loginResult); err != nil {
		log.Fatalf("auth.login failed: %v", err)
	}
	if !loginResult {
		log.Fatal("auth.login returned false")
	}
	log.Println("Logged in to TrueNAS API")

	// Create API key
	type apiKeyCreateParams struct {
		Name     string `json:"name"`
		Username string `json:"username"`
	}
	type apiKeyResult struct {
		ID  int64  `json:"id"`
		Key string `json:"key"`
	}

	var keyResult apiKeyResult
	if err := call(apiConn, "api_key.create", []any{apiKeyCreateParams{Name: *apiKeyName, Username: "truenas_admin"}}, &keyResult); err != nil {
		log.Fatalf("api_key.create failed: %v", err)
	}

	log.Printf("Created API key (id=%d)", keyResult.ID)

	// Write to file
	if err := os.WriteFile(*outputFile, []byte(keyResult.Key), 0600); err != nil {
		log.Fatalf("Failed to write API key to %s: %v", *outputFile, err)
	}

	fmt.Println(keyResult.Key)
	log.Printf("API key written to %s", *outputFile)

	// Phase 3: Create data pool
	log.Println("=== Phase 3: Create Data Pool ===")

	// Get all disks from the system
	var apiDisks []map[string]any
	if err := call(apiConn, "disk.query", nil, &apiDisks); err != nil {
		log.Fatalf("disk.query failed: %v", err)
	}

	// Get the boot disk(s) to exclude them
	var bootDiskNames []string
	if err := call(apiConn, "boot.get_disks", nil, &bootDiskNames); err != nil {
		log.Fatalf("boot.get_disks failed: %v", err)
	}

	bootDisks := map[string]bool{}
	for _, name := range bootDiskNames {
		bootDisks[name] = true
	}
	log.Printf("Boot disks: %v", bootDiskNames)

	var poolDisks []string
	for _, d := range apiDisks {
		name, ok := d["name"].(string)
		if !ok {
			continue
		}
		if bootDisks[name] {
			log.Printf("Skipping boot disk: %s", name)
			continue
		}
		poolDisks = append(poolDisks, name)
	}

	if len(poolDisks) == 0 {
		log.Fatal("No available disks for data pool. The VM needs at least 2 disks (1 for OS, 1+ for data).")
	}

	log.Printf("Available disks for pool: %v", poolDisks)

	topology := map[string]any{
		"data": []map[string]any{
			{
				"type":  "STRIPE",
				"disks": poolDisks,
			},
		},
	}

	poolParams := map[string]any{
		"name":     *poolName,
		"topology": topology,
	}

	var jobID int64
	if err := call(apiConn, "pool.create", []any{poolParams}, &jobID); err != nil {
		log.Fatalf("pool.create failed: %v", err)
	}
	log.Printf("Pool creation job started (id=%d), waiting...", jobID)

	// pool.create sends async notifications that interfere with subsequent calls,
	// so disconnect, wait, and reconnect to poll for pool readiness.
	apiConn.Close()
	time.Sleep(5 * time.Second)

	apiURL2 := fmt.Sprintf("wss://%s:%d/api/current", *host, *httpsPort)
	apiConn2, err := waitForWebSocket(apiURL2, wssDialer, 2*time.Minute)
	if err != nil {
		log.Fatalf("Failed to reconnect to TrueNAS API: %v", err)
	}
	defer apiConn2.Close()

	var loginResult2 bool
	if err := call(apiConn2, "auth.login", []any{"truenas_admin", *adminPassword}, &loginResult2); err != nil {
		log.Fatalf("auth.login on reconnect failed: %v", err)
	}

	deadline := time.Now().Add(5 * time.Minute)
	for time.Now().Before(deadline) {
		var pools []map[string]any
		if err := call(apiConn2, "pool.query", nil, &pools); err != nil {
			log.Printf("pool.query failed (retrying): %v", err)
			time.Sleep(3 * time.Second)
			continue
		}
		for _, p := range pools {
			if name, ok := p["name"].(string); ok && name == *poolName {
				log.Printf("Created pool %q", *poolName)
				goto poolReady
			}
		}
		log.Printf("Pool not yet available (got %d pools), retrying...", len(pools))
		time.Sleep(3 * time.Second)
	}
	log.Fatalf("Timed out waiting for pool %q to be created", *poolName)
poolReady:
	log.Println("Setup complete!")
}
