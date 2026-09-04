import { useEffect, useRef, useState } from 'react';
import { useParams } from 'react-router-dom';
import { api } from '../api';
import Dialog from '../components/Dialog';
import ConfirmDialog from '../components/ConfirmDialog';
import { useDialog } from '../hooks/useDialog';

export default function CampaignDetail() {
  const { id } = useParams();
  const [campaign, setCampaign] = useState(null);
  const {
    dialog, confirm, showError, showConfirm,
    closeDialog, handleConfirm, handleConfirmCancel,
  } = useDialog();
  const pollErrorShown = useRef(false);

  useEffect(() => {
    let active = true;
    async function poll() {
      try {
        const c = await api.campaigns.get(id);
        if (active) {
          setCampaign(c);
          pollErrorShown.current = false;
        }
      } catch (e) {
        if (active && !pollErrorShown.current) {
          pollErrorShown.current = true;
          showError(e.message, 'Failed to Load Campaign');
        }
      }
    }
    poll();
    const interval = setInterval(poll, 4000);
    return () => { active = false; clearInterval(interval); };
  }, [id]);

  async function handleCancel() {
    const confirmed = await showConfirm(
      'No further emails will be sent for this campaign.',
      'Cancel Campaign',
      'Cancel Campaign',
    );
    if (!confirmed) return;
    try {
      await api.campaigns.cancel(id);
      const c = await api.campaigns.get(id);
      setCampaign(c);
    } catch (e) {
      showError(e.message, 'Cancel Failed');
    }
  }

  if (!campaign) return <div className="layout">Loading...</div>;

  const pct = campaign.total > 0 ? Math.round(((campaign.sent + campaign.failed) / campaign.total) * 100) : 0;
  const canCancel = ['SCHEDULED', 'RUNNING'].includes(campaign.status);

  return (
    <div className="layout">
      <div className="card">
        <h2>Campaign #{campaign.campaignNumber}</h2>
        <span className={`status status-${campaign.status}`}>{campaign.status}</span>

        <div className="progress-bar" style={{ marginTop: 16 }}>
          <div className="progress-bar-fill" style={{ width: `${pct}%` }} />
        </div>

        <p><strong className="text-emerald">{campaign.sent}</strong> / {campaign.total} sent</p>
        {campaign.failed > 0 && <p className="error">{campaign.failed} failed</p>}

        <p className="muted">
          Scheduled: {new Date(campaign.scheduledAt).toLocaleString()}
          {campaign.startedAt && <> · Started: {new Date(campaign.startedAt).toLocaleString()}</>}
          {campaign.finishedAt && <> · Finished: {new Date(campaign.finishedAt).toLocaleString()}</>}
        </p>

        {canCancel && (
          <button type="button" className="btn btn-danger" onClick={handleCancel}>Cancel Campaign</button>
        )}
      </div>

      <Dialog {...dialog} onClose={closeDialog} />
      <ConfirmDialog confirm={confirm} onConfirm={handleConfirm} onClose={handleConfirmCancel} />
    </div>
  );
}
