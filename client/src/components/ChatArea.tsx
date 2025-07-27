// src/components/ChatArea.tsx
import React from "react";
import Chat from "./Chat";

const ChatArea: React.FC = () => {
  return (
    <div
      style={{
        flex: 1,
        minHeight: 0,
        maxWidth: '90%',
        margin: "0 auto",
        padding: 16,
        background: "#fff",
        borderRadius: 8,
        boxShadow: "0 2px 8px rgba(0,0,0,0.1)",
        display: 'flex',
        flexDirection: 'column',
      }}
    >
      <Chat />
    </div>
  );
};

export default ChatArea;
