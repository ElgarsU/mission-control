package ws

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Message is the envelope for all protocol messages between agent and relay.
type Message struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data,omitempty"`
}

// --- Agent → Relay ---

type SessionCreatedData struct {
	SessionID string    `json:"session_id"`
	Project   string    `json:"project"`
	CreatedAt time.Time `json:"created_at"`
}

type SessionClosedData struct {
	SessionID string `json:"session_id"`
}

type SessionListEntry struct {
	SessionID string    `json:"session_id"`
	Project   string    `json:"project"`
	CreatedAt time.Time `json:"created_at"`
}

type SessionListData struct {
	Sessions []SessionListEntry `json:"sessions"`
}

// --- Relay → Agent ---

type SessionCreateData struct {
	Project       string `json:"project"`
	InitialPrompt string `json:"initial_prompt,omitempty"`
	WorkingDir    string `json:"working_dir,omitempty"`
}

type SessionInputData struct {
	SessionID string `json:"session_id"`
	Content   string `json:"content"`
}

type SessionKillData struct {
	SessionID string `json:"session_id"`
}

// Client is a WebSocket client that connects to the relay server.
type Client struct {
	url    string
	conn   *websocket.Conn
	mu     sync.Mutex
	closed bool
}

// NewClient creates a new WebSocket client. Call Connect() to establish the connection.
func NewClient(url string) *Client {
	return &Client{url: url}
}

// Connect dials the relay server. Authentication is handled by WireGuard at the network layer.
func (c *Client) Connect() error {
	conn, _, err := websocket.DefaultDialer.Dial(c.url, nil)
	if err != nil {
		return fmt.Errorf("ws connect: %w", err)
	}

	c.mu.Lock()
	c.conn = conn
	c.closed = false
	c.mu.Unlock()

	return nil
}

// Send marshals a message to JSON and sends it to the relay.
func (c *Client) Send(msg Message) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed || c.conn == nil {
		return fmt.Errorf("ws send: connection closed")
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("ws marshal: %w", err)
	}
	return c.conn.WriteMessage(websocket.TextMessage, data)
}

// Close shuts down the WebSocket connection.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed || c.conn == nil {
		return nil
	}
	c.closed = true
	return c.conn.Close()
}

// NewMessage creates a Message with the given type and data, marshaling data to JSON.
func NewMessage(msgType string, data any) (Message, error) {
	raw, err := json.Marshal(data)
	if err != nil {
		return Message{}, err
	}
	return Message{Type: msgType, Data: raw}, nil
}
