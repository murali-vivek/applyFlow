import { Routes, Route, Navigate, Link } from 'react-router-dom';
import { useEffect, useState } from 'react';
import { api, googleLoginUrl, clearToken } from './api';
import AuthCallback from './pages/AuthCallback';
import Login from './pages/Login';
import Dashboard from './pages/Dashboard';
import Resume from './pages/Resume';
import Templates from './pages/Templates';
import NewCampaign from './pages/NewCampaign';
import CampaignDetail from './pages/CampaignDetail';

function ProtectedRoute({ children }) {
  const [status, setStatus] = useState('loading');

  useEffect(() => {
    api.me()
      .then(() => setStatus('ok'))
      .catch(() => setStatus('fail'));
  }, []);

  if (status === 'loading') return <div className="layout">Loading...</div>;
  if (status === 'fail') return <Navigate to="/" replace />;
  return <>{children}</>;
}

function Nav() {
  return (
    <nav className="nav">
      <Link to="/dashboard" className="brand">ApplyFlow</Link>
      <Link to="/dashboard">Dashboard</Link>
      <Link to="/resume">Resume</Link>
      <Link to="/templates">Templates</Link>
      <Link to="/campaigns/new">New Campaign</Link>
      <button className="btn btn-nav" onClick={() => { clearToken(); window.location.href = '/'; }}>Logout</button>
    </nav>
  );
}

export default function App() {
  return (
    <Routes>
      <Route path="/" element={<Login />} />
      <Route path="/auth/callback" element={<AuthCallback />} />
      <Route path="/dashboard" element={<ProtectedRoute><Nav /><Dashboard /></ProtectedRoute>} />
      <Route path="/resume" element={<ProtectedRoute><Nav /><Resume /></ProtectedRoute>} />
      <Route path="/templates" element={<ProtectedRoute><Nav /><Templates /></ProtectedRoute>} />
      <Route path="/campaigns/new" element={<ProtectedRoute><Nav /><NewCampaign /></ProtectedRoute>} />
      <Route path="/campaigns/:id" element={<ProtectedRoute><Nav /><CampaignDetail /></ProtectedRoute>} />
    </Routes>
  );
}
