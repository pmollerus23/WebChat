// src/components/Navbar.tsx
import React from 'react';
import { AppBar, Toolbar, Typography } from '@mui/material';

const Navbar: React.FC = () => {
  return (
    <AppBar position="static">
      <Toolbar sx={{
        backgroundColor: '#581845'
      }}>
        <Typography variant="h6" component="div">
          FlipNet WebChat
        </Typography>
      </Toolbar>
    </AppBar>
  );
};

export default Navbar;
