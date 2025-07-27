// src/pages/Home.tsx
import React from 'react';
import MainLayout from '../layouts/MainLayout';
import ChatArea from '../components/ChatArea';

const Home: React.FC = () => {
  return (
    <MainLayout>
      <ChatArea />
    </MainLayout>
  );
};

export default Home;
