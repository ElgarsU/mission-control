package ws_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"mission-control/mc-agent/internal/ws"
)

var upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

// wsURL converts an httptest server URL from http:// to ws://.
func wsURL(s *httptest.Server) string {
	return "ws" + strings.TrimPrefix(s.URL, "http")
}

// Connect to unreachable server returns error.
func TestConnect_unreachableServer_returnsError(t *testing.T) {
	client := ws.NewClient("ws://127.0.0.1:1")
	err := client.Connect()
	if err == nil {
		t.Error("expected error connecting to unreachable server, got nil")
		client.Close()
	}
}

// Connect establishes a connection that the server sees.
func TestConnect_serverSeesConnection(t *testing.T) {
	connected := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		close(connected)
		conn.ReadMessage()
	}))
	defer srv.Close()

	client := ws.NewClient(wsURL(srv))
	if err := client.Connect(); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer client.Close()

	select {
	case <-connected:
	case <-time.After(2 * time.Second):
		t.Fatal("server didn't see connection")
	}
}

// Send transmits a JSON message that the server can read.
func TestSend_serverReceivesMessage(t *testing.T) {
	received := make(chan ws.Message, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		_, data, err := conn.ReadMessage()
		if err != nil {
			return
		}
		var msg ws.Message
		json.Unmarshal(data, &msg)
		received <- msg
	}))
	defer srv.Close()

	client := ws.NewClient(wsURL(srv))
	if err := client.Connect(); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer client.Close()

	msg, err := ws.NewMessage("session.created", ws.SessionCreatedData{
		SessionID: "abc123",
		Project:   "test-proj",
		CreatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("NewMessage: %v", err)
	}
	if err := client.Send(msg); err != nil {
		t.Fatalf("Send: %v", err)
	}

	select {
	case msg := <-received:
		if msg.Type != "session.created" {
			t.Errorf("type = %q, want %q", msg.Type, "session.created")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for server to receive message")
	}
}

// Send on a closed client returns error.
func TestSend_afterClose_returnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		conn.ReadMessage()
	}))
	defer srv.Close()

	client := ws.NewClient(wsURL(srv))
	if err := client.Connect(); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	client.Close()

	err := client.Send(ws.Message{Type: "session.list"})
	if err == nil {
		t.Error("expected error sending on closed client, got nil")
	}
}

// Close cleanly shuts down the connection.
func TestClose_shutsDownCleanly(t *testing.T) {
	serverClosed := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		conn.ReadMessage()
		close(serverClosed)
	}))
	defer srv.Close()

	client := ws.NewClient(wsURL(srv))
	if err := client.Connect(); err != nil {
		t.Fatalf("Connect: %v", err)
	}

	err := client.Close()
	if err != nil {
		t.Errorf("Close: %v", err)
	}

	select {
	case <-serverClosed:
	case <-time.After(2 * time.Second):
		t.Error("server didn't detect client disconnect")
	}
}

// Double close doesn't panic.
func TestClose_twice_noPanic(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		conn.ReadMessage()
	}))
	defer srv.Close()

	client := ws.NewClient(wsURL(srv))
	if err := client.Connect(); err != nil {
		t.Fatalf("Connect: %v", err)
	}

	client.Close()
	client.Close() // should not panic
}
