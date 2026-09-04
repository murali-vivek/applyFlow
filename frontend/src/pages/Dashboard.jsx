import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { api } from '../api';
import CampaignList from '../components/CampaignList';
import Dialog from '../components/Dialog';
import { useDialog } from '../hooks/useDialog';

export default function Dashboard() {
  const [campaigns, setCampaigns] = useState([]);
  const { dialog, showError, closeDialog } = useDialog();

  useEffect(() => {
    api.campaigns.list()
      .then((data) => setCampaigns(data ?? []))
      .catch((e) => showError(e.message, 'Failed to Load Campaigns'));
  }, []);

  const totalSent = campaigns.reduce((s, c) => s + c.sent, 0);
  const totalCompleted = campaigns.filter((c) => c.status === 'COMPLETED').length;
  const totalActive = campaigns.filter((c) => ['RUNNING', 'SCHEDULED'].includes(c.status)).length;

  return (
    <div className="layout">
      {campaigns.length > 0 && (
        <div className="stats-row">
          <div className="stat-card stat-emerald">
            <div className="stat-value">{totalSent}</div>
            <div className="stat-label">Emails Sent</div>
          </div>
          <div className="stat-card stat-navy">
            <div className="stat-value">{totalActive}</div>
            <div className="stat-label">Active Campaigns</div>
          </div>
          <div className="stat-card stat-gold">
            <div className="stat-value">{totalCompleted}</div>
            <div className="stat-label">Completed</div>
          </div>
        </div>
      )}
      <div className="card">
        <div className="card-header-row">
          <h2>Campaigns</h2>
          <Link to="/campaigns/new" className="btn btn-primary">New Campaign</Link>
        </div>
        {campaigns.length === 0 ? (
          <p className="muted">No campaigns yet. <Link to="/campaigns/new">Create one</Link></p>
        ) : (
          <CampaignList campaigns={campaigns} />
        )}
      </div>

      <Dialog {...dialog} onClose={closeDialog} />
    </div>
  );
}
