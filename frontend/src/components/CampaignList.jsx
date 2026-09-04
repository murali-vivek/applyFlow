import { Link } from 'react-router-dom';

export default function CampaignList({ campaigns, compact = false }) {
  if (!campaigns.length) {
    return <p className="muted">No campaigns yet.</p>;
  }

  return (
    <table className={compact ? 'campaign-table-compact' : ''}>
      <thead>
        <tr>
          <th>#</th>
          <th>Status</th>
          <th>Progress</th>
          {!compact && <th>Scheduled</th>}
          <th></th>
        </tr>
      </thead>
      <tbody>
        {campaigns.map((c) => (
          <tr key={c.id}>
            <td>#{c.campaignNumber}</td>
            <td><span className={`status status-${c.status}`}>{c.status}</span></td>
            <td>{c.sent} / {c.total} sent{c.failed > 0 ? `, ${c.failed} failed` : ''}</td>
            {!compact && <td>{new Date(c.scheduledAt).toLocaleString()}</td>}
            <td>
              <Link to={`/campaigns/${c.id}`} className="btn btn-view">
                View Campaign
              </Link>
            </td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}
