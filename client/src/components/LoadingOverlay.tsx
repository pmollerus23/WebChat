import React from "react";
import { CircularProgress, Box } from "@mui/material";
import { useChat } from "../context/ChatContext";

const overlayColor = "rgba(245, 245, 245, 0.56)"; // transluscent light grey
const spinnerColor = "#581845"; // matches NavBar maroon

const LoadingOverlay: React.FC = () => {
  const { loading } = useChat();

  if (!loading) return null;

  return (
    <Box
      sx={{
        position: "fixed",
        zIndex: 1300,
        top: 0,
        left: 0,
        width: "100vw",
        height: "100vh",
        bgcolor: overlayColor,
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
      }}
    >
      <CircularProgress size={80} thickness={5} sx={{ color: spinnerColor }} />
    </Box>
  );
};

export default LoadingOverlay;
