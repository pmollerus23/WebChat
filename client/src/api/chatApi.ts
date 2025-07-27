
import type { ChatMessage } from "../types/ChatMessage";

// Fetch the last 100 messages from the backend
export async function fetchMessages(): Promise<ChatMessage[]> {
  const apiUrl = import.meta.env.VITE_API_URL as string | undefined;
  if (!apiUrl) {
	  throw new Error("API URL KEY not defined in .env");
  }
  const res = await fetch(apiUrl);
  if (!res.ok) throw new Error("Failed to fetch messages");
  return res.json();
}

// Connect to the WebSocket server
export function connectWebSocket(onMessage: (msg: ChatMessage) => void): WebSocket {
  // Connect directly to the Go server WebSocket endpoint
  const wsUrl = import.meta.env.VITE_WS_URL as string | undefined;
  if (!wsUrl){
	  throw new Error("WS URL KEY not defined in .env");
  }
  const ws = new WebSocket(wsUrl);

  ws.onmessage = (event) => {
    try {
      const msg: ChatMessage = JSON.parse(event.data);
      onMessage(msg);
    } catch (e) {
      // Optionally handle parse errors
      // console.error("Invalid WS message", e);
    }
  };
  return ws;
}

// Send a message over the WebSocket
export function sendMessage(ws: WebSocket, msg: Omit<ChatMessage, "id" | "timestamp"> & { timestamp?: string }) {
  // The server will assign timestamp and id, but we can send name/content
  ws.send(JSON.stringify({
    name: msg.name,
    content: msg.content,
    // Optionally send timestamp if you want client-side time
    ...(msg.timestamp ? { timestamp: msg.timestamp } : {}),
  }));
}
