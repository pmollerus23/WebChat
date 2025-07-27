// src/layouts/MainLayout.tsx
import React from 'react';
import Navbar from '../components/NavBar';
import { Container, Box } from '@mui/material';

interface MainLayoutProps {
  children: React.ReactNode;
}

const MainLayout: React.FC<MainLayoutProps> = ({ children }) => {
  return (
    <Box sx={{ minHeight: '100vh', height: '100vh', backgroundColor: 'beige', display: 'flex', flexDirection: 'column' }}>
      <Navbar />
      <Container sx={{ flex: 1, display: 'flex', flexDirection: 'column', minHeight: 0, mt: 4, pb: 4 }}>
        {children}
      </Container>
    </Box>
  );
};

export default MainLayout;
