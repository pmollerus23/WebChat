import React, { createContext, useContext, useState } from 'react';
import type { ReactNode } from 'react';
import type { ChatMessage } from '../types/ChatMessage'; // Assume you have this defined
import { getOrCreateGuestId } from '../Utils';

interface ChatContextType {
  messages: ChatMessage[];
  loading: boolean;
  error: string | null;
  name: string;

  setName: React.Dispatch<React.SetStateAction<string>>;
  setLoading: React.Dispatch<React.SetStateAction<boolean>>;
  setMessages: React.Dispatch<React.SetStateAction<ChatMessage[]>>;
  setError: React.Dispatch<React.SetStateAction<string | null>>;
}

const ChatContext = createContext<ChatContextType | undefined>(undefined);

export const useChat = () => {
  const context = useContext(ChatContext);
  if (!context) throw new Error('useChat must be used within ChatProvider');
  return context;
};

interface ChatProviderProps {
  children: ReactNode;
}

export const ChatProvider: React.FC<ChatProviderProps> = ({ children }) => {
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // TODO - Set guestName in Chat.tsx
  const [name, setName] = useState<string>(() => `guest${getOrCreateGuestId()}`); // Default name




  return (
    <ChatContext.Provider value={{ messages, loading, error, name, setName, setMessages, setLoading, setError }}>
      {children}
    </ChatContext.Provider>
  );
};
