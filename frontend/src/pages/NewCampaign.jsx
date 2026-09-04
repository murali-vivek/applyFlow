import { useEffect, useRef, useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { api } from '../api';
import CampaignList from '../components/CampaignList';
import Dialog from '../components/Dialog';
import DateTimePicker from '../components/DateTimePicker';
import XlsxPreview from '../components/XlsxPreview';
import { useDialog } from '../hooks/useDialog';
import { buildUploadSummary, hasUploadWarnings } from '../utils/uploadSummary';

export default function NewCampaign() {
  const navigate = useNavigate();
  const fileInputRef = useRef(null);
  const [templates, setTemplates] = useState([]);
  const [campaigns, setCampaigns] = useState([]);
  const [templateId, setTemplateId] = useState('');
  const [xlsxResult, setXlsxResult] = useState(null);
  const [startAt, setStartAt] = useState('');
  const [startImmediately, setStartImmediately] = useState(false);
  const [uploading, setUploading] = useState(false);
  const { dialog, showError, showSuccess, closeDialog } = useDialog();

  useEffect(() => {
    api.templates.list().then((t) => {
      setTemplates(t ?? []);
      const def = t?.find((x) => x.isDefault) || t?.[0];
      if (def) setTemplateId(def.id);
    }).catch((e) => showError(e.message, 'Failed to Load Templates'));

    api.campaigns.list()
      .then((data) => setCampaigns(data ?? []))
      .catch(() => {});

    const now = new Date();
    now.setMinutes(now.getMinutes() - now.getTimezoneOffset());
    setStartAt(now.toISOString().slice(0, 16));
  }, []);

  async function handleXlsx(e) {
    const file = e.target.files?.[0];
    e.target.value = '';
    if (!file) return;

    if (!file.name.toLowerCase().endsWith('.xlsx')) {
      showError('Only spreadsheet files (.xlsx) are allowed.', 'Invalid File');
      setXlsxResult(null);
      return;
    }

    setUploading(true);
    try {
      const result = await api.xlsx.upload(file);
      if (!result.valid) {
        const message = result.errors?.[0] || 'The uploaded file could not be used.';
        showError(message, 'Spreadsheet Validation Failed');
        setXlsxResult(null);
        return;
      }
      setXlsxResult(result);
      if (hasUploadWarnings(result)) {
        showSuccess(buildUploadSummary(result), 'Upload Summary');
      }
    } catch (err) {
      showError(err.message, 'Upload Failed');
      setXlsxResult(null);
    } finally {
      setUploading(false);
    }
  }

  async function handleStart() {
    if (!xlsxResult?.id || !templateId) {
      showError('Upload a company list and select a template.', 'Missing Information');
      return;
    }
    try {
      const campaign = await api.campaigns.create({
        xlsxFileId: xlsxResult.id,
        templateId,
        startAt: startImmediately ? new Date().toISOString() : new Date(startAt).toISOString(),
      });
      navigate(`/campaigns/${campaign.id}`);
    } catch (err) {
      showError(err.message, 'Campaign Error');
    }
  }

  return (
    <div className="layout">
      <div className="card">
        <div className="card-header-row">
          <h2>New Campaign</h2>
          <Link to="/dashboard" className="btn btn-secondary">View All Campaigns</Link>
        </div>
        <p className="muted">Upload a spreadsheet with columns: Company Name, Role, Company Mail (max 400 rows).</p>

        <div className="file-upload-section">
          <label className="field-label">Company List</label>
          <div className="file-upload-row">
            <input
              ref={fileInputRef}
              type="file"
              accept=".xlsx"
              onChange={handleXlsx}
              hidden
            />
            <button
              type="button"
              className={`btn ${xlsxResult?.valid ? 'btn-file-chosen' : 'btn-primary'}`}
              disabled={uploading}
              onClick={() => fileInputRef.current?.click()}
            >
              {uploading ? 'Validating…' : xlsxResult?.valid ? 'Change File' : 'Upload Company List'}
            </button>
          </div>

          {xlsxResult?.valid && (
            <XlsxPreview
              fileName={xlsxResult.fileName}
              rowCount={xlsxResult.rowCount}
              preview={xlsxResult.preview}
            />
          )}
        </div>

        <label className="field-label">Template</label>
        <select value={templateId} onChange={(e) => setTemplateId(e.target.value)}>
          {templates.map((t) => (
            <option key={t.id} value={t.id}>{t.name}{t.isDefault ? ' (default)' : ''}</option>
          ))}
        </select>

        <label className="field-label">Start At</label>
        <DateTimePicker value={startAt} onChange={setStartAt} disabled={startImmediately} />

        <label className="checkbox-row schedule-option">
          <input
            type="checkbox"
            checked={startImmediately}
            onChange={(e) => setStartImmediately(e.target.checked)}
          />
          Start immediately
        </label>

        <button type="button" className="btn btn-primary" onClick={handleStart} style={{ marginTop: 16 }}>
          Start Campaign
        </button>
      </div>

      {campaigns.length > 0 && (
        <div className="card">
          <h3>Your Existing Campaigns</h3>
          <CampaignList campaigns={campaigns} compact />
        </div>
      )}

      <Dialog {...dialog} onClose={closeDialog} />
    </div>
  );
}
