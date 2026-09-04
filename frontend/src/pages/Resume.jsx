import { useEffect, useRef, useState } from 'react';
import { api } from '../api';
import Dialog from '../components/Dialog';
import { useDialog } from '../hooks/useDialog';

const MAX_SIZE_BYTES = 10 * 1024 * 1024; // 10 MB

function validateFile(file) {
  if (!file) return 'No file selected.';
  const name = file.name.toLowerCase();
  if (!name.endsWith('.pdf')) {
    return 'Only PDF files are allowed.';
  }
  if (file.type && file.type !== 'application/pdf') {
    return 'Only PDF files are allowed.';
  }
  if (file.size === 0) {
    return 'The selected file is empty.';
  }
  if (file.size > MAX_SIZE_BYTES) {
    return 'File exceeds the 10 MB limit.';
  }
  return null;
}

export default function Resume() {
  const fileInputRef = useRef(null);
  const [resume, setResume] = useState(null);
  const [uploading, setUploading] = useState(false);
  const { dialog, showError, showSuccess, closeDialog } = useDialog();

  useEffect(() => {
    api.resume.get().then((r) => setResume(r)).catch(() => {});
  }, []);

  async function handleUpload(e) {
    const file = e.target.files?.[0];
    e.target.value = '';
    if (!file) return;

    const validationError = validateFile(file);
    if (validationError) {
      showError(validationError, 'Upload Failed');
      return;
    }

    setUploading(true);
    try {
      const r = await api.resume.upload(file);
      setResume(r);
      showSuccess('Your resume was uploaded successfully and is ready to use in campaigns.', 'Resume Uploaded');
    } catch (err) {
      showError(err.message, 'Upload Failed');
    } finally {
      setUploading(false);
    }
  }

  return (
    <div className="layout">
      <div className="card">
        <h2>Resume</h2>
        <p className="muted">Upload your standard resume (PDF only, max 10 MB). One active resume per account.</p>

        {resume && (
          <div className="current-file">
            <span className="current-file-label">Current:</span>
            <strong>{resume.name}</strong>
            <span className="muted"> (uploaded {new Date(resume.uploadedAt).toLocaleString()})</span>
          </div>
        )}

        <div className="file-upload-row">
          <input
            ref={fileInputRef}
            type="file"
            accept=".pdf,application/pdf"
            onChange={handleUpload}
            hidden
          />
          <button
            type="button"
            className="btn btn-primary"
            disabled={uploading}
            onClick={() => fileInputRef.current?.click()}
          >
            {uploading ? 'Uploading…' : 'Choose File'}
          </button>
        </div>
      </div>

      <Dialog {...dialog} onClose={closeDialog} />
    </div>
  );
}
