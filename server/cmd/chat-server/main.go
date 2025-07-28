package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5/pgxpool"
)

// config.go
type Config struct {
	DBUrl         string
	Port          string
	AllowedOrigin string
}

var config Config

// ---------------- DB Setup ----------------

var db *pgxpool.Pool

type Message struct {
	ID        int       `json:"id,omitempty"`
	Name      string    `json:"name"` // NEW: username or display name
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

func saveMessage(msg Message) {
	_, err := db.Exec(context.Background(),
		"INSERT INTO messages (name, content, timestamp) VALUES ($1, $2, $3)",
		msg.Name, msg.Content, msg.Timestamp,
	)
	if err != nil {
		log.Println("DB insert error:", err)
	}
}

func getRecentMessages(limit int) ([]Message, error) {
	rows, err := db.Query(context.Background(), `
		SELECT id, name, content, timestamp
        FROM messages
        ORDER BY timestamp DESC
        LIMIT $1
		`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var msg Message
		if err := rows.Scan(&msg.ID, &msg.Name, &msg.Content, &msg.Timestamp); err != nil {
			continue
		}
		messages = append(messages, msg)
	}
	return messages, nil
}

// Helper to get real client IP from a request
func getRealIP(r *http.Request) string {
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		// Could be comma separated list
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}
	// fallback to remote addr (includes port)
	return r.RemoteAddr
}

// ---------------- WebSocket ----------------

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Client struct {
	conn *websocket.Conn
	send chan []byte
}

type Hub struct {
	clients    map[*Client]bool
	mu         sync.Mutex // Add this line
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
}

func newHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
		case message := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
		}
	}
}

func (c *Client) readPump(h *Hub) {
	defer func() {
		h.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(512)
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second)) // extend deadline
		return nil
	})

	for {
		_, rawMsg, err := c.conn.ReadMessage()
		if err != nil {
			log.Println("read error:", err)
			break
		}

		var msg Message
		if err := json.Unmarshal(rawMsg, &msg); err != nil {
			log.Println("invalid message format:", err)
			continue
		}
		msg.Timestamp = time.Now().UTC()
		go saveMessage(msg)

		encoded, _ := json.Marshal(msg)
		h.broadcast <- encoded
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(30 * time.Second) // Adjust as needed
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			if !ok {
				// Channel closed, send close message
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			err := c.conn.WriteMessage(websocket.TextMessage, msg)
			if err != nil {
				log.Println("write error:", err)
				return
			}
		case <-ticker.C:
			// Send Ping
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Println("ping error:", err)
				return
			}
		}
	}
}

func serveWs(h *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("upgrade error:", err)
		return
	}
	client := &Client{conn: conn, send: make(chan []byte, 256)}
	h.register <- client

	go client.writePump()
	client.readPump(h)
}

// ---------------- HTTP API ----------------

// ----------------HTTP Middleware -----------

// loggingMiddleware logs incoming HTTP requests and the time taken to process them
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap ResponseWriter to capture status code and response body
		lrw := &loggingResponseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
			body:           &bytes.Buffer{},
		}
		next.ServeHTTP(lrw, r)

		duration := time.Since(start)

		// Log the request with basic info
		logMsg := fmt.Sprintf("%s %s %s %d %s", getRealIP(r), r.Method, r.URL.Path, lrw.statusCode, duration)

		// If there's an error status code, also log the response body
		if lrw.statusCode >= 400 {
			logMsg += fmt.Sprintf(" - Error: %s", lrw.body.String())
		}

		log.Printf(logMsg)
	})
}

type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
	body       *bytes.Buffer
}

// WriteHeader captures status code
func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

// Write captures the response body for error logging
func (lrw *loggingResponseWriter) Write(data []byte) (int, error) {
	// Only capture body for error responses to avoid memory issues
	if lrw.statusCode >= 400 {
		lrw.body.Write(data)
	}
	return lrw.ResponseWriter.Write(data)
}

// Hijack delegates to the underlying ResponseWriter if it supports Hijacker
func (lrw *loggingResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hj, ok := lrw.ResponseWriter.(http.Hijacker); ok {
		return hj.Hijack()
	}
	return nil, nil, fmt.Errorf("ResponseWriter does not implement http.Hijacker")
}

// Enhanced error handling in your handlers (RECOMMENDED APPROACH)
func handleGetMessages(w http.ResponseWriter, r *http.Request) {
	messages, err := getRecentMessages(100)
	if err != nil {
		// Log the actual error details with context
		log.Printf("ERROR in handleGetMessages - getRecentMessages failed: %v", err)

		// Return error to client
		http.Error(w, "Failed to get messages", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(messages); err != nil {
		log.Printf("ERROR in handleGetMessages - JSON encoding failed: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

func wsAuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("WebSocket connection from %s", getRealIP(r))
		next(w, r)
	}
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", config.AllowedOrigin) // Or specify your frontend URL instead of "*"
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (h *Hub) Shutdown() {
	// Your logic to close all connections, e.g.
	h.mu.Lock()
	defer h.mu.Unlock()
	for c := range h.clients {
		close(c.send) // or whatever method closes your client socket
	}
}

// ---------------- Main ----------------

func main() {

	var err error

	// Populate config
	config = Config{
		DBUrl:         os.Getenv("DATABASE_URL"),
		Port:          os.Getenv("PORT"),
		AllowedOrigin: os.Getenv("ALLOWED_ORIGIN"),
	}

	if config.Port == "" {
		config.Port = "8080" // default fallback
	}

	// Connect to DB
	db, err = pgxpool.New(context.Background(), config.DBUrl)
	if err != nil {
		log.Fatal("DB connection failed:", err)
	}

	// defer func() {
	// 	log.Println("closing db connection ...")
	// 	db.Close()
	// }()

	// Test the connection
	if err := db.Ping(context.Background()); err != nil {
		log.Fatal("DB connection test failed:", err)
	}

	// Ensure table exists
	_, err = db.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS messages (
			id SERIAL PRIMARY KEY,
			name TEXT NULL,
			content TEXT NOT NULL,
			timestamp TIMESTAMPTZ NOT NULL
		)
	`)
	if err != nil {
		log.Fatal("Failed to create messages table:", err)
	}

	// Set up hub
	hub := newHub()
	go hub.run()

	// Set up HTTP server
	srv := &http.Server{
		Addr:    "0.0.0.0:" + config.Port,
		Handler: nil, // Use default handler
	}

	// Prepare graceful shutdown context
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	// Defer all cleanup in one function
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Panic during shutdown: %v", r)
		}
		log.Println("Shutting down HTTP server ...")
		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("HTTP server Shutdown: %v", err)
		}
		log.Println("HTTP server shutdown complete.")

		log.Println("Closing client connections ...")
		hub.Shutdown()
		log.Println("Client connections closed.")

		log.Println("Closing DB connection ...")
		db.Close()
		log.Println("DB connection closed")
		cancel()
		log.Println("Gracefully shutdown!")
	}()

	// Routes with logging middleware
	http.Handle("/ws", loggingMiddleware(wsAuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		serveWs(hub, w, r)
	})))
	http.Handle("/api/messages", corsMiddleware(loggingMiddleware(http.HandlerFunc(handleGetMessages))))

	// Force IPv4 listener
	l, err := net.Listen("tcp4", "0.0.0.0:"+config.Port)
	if err != nil {
		log.Fatal("Failed to bind IPv4:", err)
	}

	// Channel for OS signals
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Start the server in a goroutine
	go func() {
		log.Println("Server started on 0.0.0.0:" + config.Port)
		log.Println("chat away!")
		// if err := http.Serve(l, nil); err != nil {
		if err := srv.Serve(l); err != nil && err != http.ErrServerClosed {
			log.Fatal("Serve error:", err)
		}
	}()

	// Wait for a shutdown signal
	<-stop
	log.Println("Shutting down server...")
}
