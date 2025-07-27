import { BrowserRouter, Routes, Route } from "react-router-dom";
import Home from "./pages/Home";
import { ChatProvider } from "./context/ChatContext";
import LoadingOverlay from "./components/LoadingOverlay";

export default function App() {
  return (
    <ChatProvider>
      <LoadingOverlay />
      <BrowserRouter>
        <Routes>
          <Route path="/" element={<Home />} />
        </Routes>
      </BrowserRouter>
    </ChatProvider>
  );
}
