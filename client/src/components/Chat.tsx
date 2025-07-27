import React, { useEffect, useRef, useState } from "react";
import type { FormEvent } from "react";
import { useChat } from "../context/ChatContext";
import { connectWebSocket, sendMessage, fetchMessages } from "../api/chatApi";
import type { ChatMessage } from "../types/ChatMessage";

interface ChatProps {
  children?: React.ReactNode;
}

const Chat: React.FC<ChatProps> = () => {
  const { messages, name, loading, error, setLoading, setMessages, setError } = useChat();
  const wsRef = useRef<WebSocket | null>(null);

  // WebSocket connection and message handling
  useEffect(() => {
    wsRef.current = connectWebSocket((newMessage: ChatMessage) => {
      addMessage(newMessage);
    });

    wsRef.current.onerror = () => {
      console.error("WebSocket error");
    };

    return () => {
      wsRef.current?.close();
    };
    // Only run once on mount/unmount
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Send a message to the server
  const handleSendMessage = (msg: Omit<ChatMessage, "id" | "timestamp"> & { timestamp?: string }) => {
    if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
      sendMessage(wsRef.current, msg);
    }
  };

  // UI state for input
  const [input, setInput] = useState("");

  // Ref for auto-scrolling
  const messagesEndRef = useRef<HTMLDivElement | null>(null);

  const handleSubmit = (e: FormEvent) => {
    e.preventDefault();
    if (input.trim()) {
      handleSendMessage({ name, content: input });
      setInput("");
    }
  };

  // Auto-scroll to bottom when messages change
  useEffect(() => {
    if (messagesEndRef.current) {
      messagesEndRef.current.scrollIntoView({ behavior: "smooth" });
    }
  }, [messages]);

    // Fetch initial messages from backend using chatApi
  useEffect(() => {
    setLoading(true);
    setError(null);
    fetchMessages()
      .then((data) => {
        setMessages(data);
      })
      .catch((err) => {
        setError(err.message || 'Unknown error');
      })
      .finally(() => {
        setLoading(false);
      });
  }, []);

  // WebSocket connection is now managed in the Chat component

  const addMessage = (msg: ChatMessage) => {
    setMessages((prev) => {
      const safePrev = Array.isArray(prev) ? prev : [];
      // Deduplicate by id if present, else by content+timestamp+name
      const exists = safePrev.some(
        m => (msg.id && m.id === msg.id) ||
              (!msg.id && m.content === msg.content && m.timestamp === msg.timestamp && m.name === msg.name)
      );
      if (exists) return safePrev;
      return [...safePrev, msg];
    });
    // You might also want to send the message to backend here
  };

//  const clearMessages = () => {
//    setMessages([]);
//  };

  // Sort messages ascending (oldest at top, newest at bottom)
  const sortedMessages = Array.isArray(messages)
    ? [...messages].sort((a, b) => {
        const ta = a.timestamp ? new Date(a.timestamp).getTime() : 0;
        const tb = b.timestamp ? new Date(b.timestamp).getTime() : 0;
        return ta - tb;
      })
    : [];

  return (
    <div style={{ display: "flex", flexDirection: "column", flex: 1, minHeight: 0, border: "1px solid #ccc", borderRadius: 8, maxWidth: 500, margin: "0 auto", background: "#fafbfc" }}>
      <div style={{ flex: 1, minHeight: 0, overflowY: "auto", padding: 16, display: "flex", flexDirection: "column", gap: 8 }}>
        {loading ? (
          <div>Loading...</div>
        ) : error ? (
          <div style={{ color: "red" }}>{error}</div>
        ) : sortedMessages.length === 0 ? (
          <div style={{ color: "#888" }}>No messages yet.</div>
        ) : (
          sortedMessages.map((msg, idx) => {
            const isOwn = msg.name === name;
            const isLast = idx === sortedMessages.length - 1;
            return (
              <div
                key={msg.id ?? idx}
                ref={isLast ? messagesEndRef : undefined}
                style={{ display: "flex", alignItems: "center", justifyContent: "space-between", background: isOwn ? "#ffeaea" : "#fff", borderRadius: 6, padding: "6px 12px" }}
              >
                <div>
                  <span style={{ fontWeight: "bold", color: isOwn ? "#d32f2f" : undefined }}>{msg.name}</span>
                  {": "}
                  <span>{msg.content}</span>
                </div>
                <span style={{ fontSize: 12, color: "#888", marginLeft: 12, whiteSpace: "nowrap" }}>{msg.timestamp ? new Date(msg.timestamp).toLocaleTimeString() : ""}</span>
              </div>
            );
          })
        )}
      </div>
      <form onSubmit={handleSubmit} style={{ display: "flex", borderTop: "1px solid #eee", padding: 8, background: "#f5f5f5" }}>
        <input
          type="text"
          value={input}
          onChange={e => setInput(e.target.value)}
          placeholder="Type your message..."
          style={{ flex: 1, padding: 8, borderRadius: 4, border: "1px solid #ccc", marginRight: 8 }}
        />
        <button type="submit" style={{ padding: "8px 16px", borderRadius: 4, border: "none", background: "#1976d2", color: "#fff", fontWeight: "bold" }}>Send</button>
      </form>
    </div>
  );
};

export default Chat;
