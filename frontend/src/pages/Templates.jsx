import { useEffect, useRef, useState } from 'react';
import { api } from '../api';
import Dialog from '../components/Dialog';
import ConfirmDialog from '../components/ConfirmDialog';
import { useDialog } from '../hooks/useDialog';

const empty = { name: '', subject: '', body: '', isDefault: false };

export default function Templates() {
  const formRef = useRef(null);
  const [templates, setTemplates] = useState([]);
  const [form, setForm] = useState(empty);
  const [editingId, setEditingId] = useState(null);
  const {
    dialog, confirm, showError, showConfirm,
    closeDialog, handleConfirm, handleConfirmCancel,
  } = useDialog();

  function load() {
    api.templates.list()
      .then((data) => setTemplates(data ?? []))
      .catch((e) => showError(e.message, 'Failed to Load Templates'));
  }

  useEffect(() => { load(); }, []);

  async function handleSubmit(e) {
    e.preventDefault();
    try {
      if (editingId) {
        await api.templates.update(editingId, form);
      } else {
        await api.templates.create(form);
      }
      setForm(empty);
      setEditingId(null);
      load();
    } catch (err) {
      showError(err.message, 'Template Error');
    }
  }

  async function handleDelete(id) {
    const confirmed = await showConfirm(
      'Are you sure you want to delete this template? This action cannot be undone.',
      'Delete Template',
      'Delete',
    );
    if (!confirmed) return;
    try {
      await api.templates.delete(id);
      load();
    } catch (err) {
      showError(err.message, 'Delete Failed');
    }
  }

  function startEdit(t) {
    setEditingId(t.id);
    setForm({ name: t.name, subject: t.subject, body: t.body, isDefault: t.isDefault });
    requestAnimationFrame(() => {
      formRef.current?.scrollIntoView({ behavior: 'smooth', block: 'start' });
    });
  }

  return (
    <div className="layout">
      <div className="card" ref={formRef}>
        <h2>{editingId ? 'Edit Template' : 'Email Templates'}</h2>
        <p className="muted">Variables: {'{{company_name}}'}, {'{{role}}'}, {'{{user_name}}'}. Max 5 templates.</p>

        <form onSubmit={handleSubmit}>
          <label className="field-label">Name</label>
          <input value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} required />
          <label className="field-label">Subject</label>
          <input value={form.subject} onChange={(e) => setForm({ ...form, subject: e.target.value })} required />
          <label className="field-label">Body</label>
          <textarea value={form.body} onChange={(e) => setForm({ ...form, body: e.target.value })} required />

          <div className="form-options">
            <label className="checkbox-row">
              <input
                type="checkbox"
                checked={form.isDefault}
                onChange={(e) => setForm({ ...form, isDefault: e.target.checked })}
              />
              <span>Set as default</span>
            </label>
          </div>

          <div className="form-actions">
            <button type="submit" className="btn btn-primary">{editingId ? 'Update' : 'Create'} Template</button>
            {editingId && (
              <button type="button" className="btn btn-secondary" onClick={() => { setEditingId(null); setForm(empty); }}>
                Cancel
              </button>
            )}
          </div>
        </form>
      </div>

      <div className="card">
        <h3>Your Templates</h3>
        {templates.map((t) => (
          <div key={t.id} className="template-item">
            <strong>{t.name}</strong>
            {t.isDefault && <span className="badge-default">Default</span>}
            <p className="muted" style={{ margin: '4px 0 10px' }}>Subject: {t.subject}</p>
            <button type="button" className="btn btn-secondary" onClick={() => startEdit(t)}>Edit</button>
            {!t.isDefault && (
              <button type="button" className="btn btn-danger" style={{ marginLeft: 8 }} onClick={() => handleDelete(t.id)}>
                Delete
              </button>
            )}
          </div>
        ))}
      </div>

      <Dialog {...dialog} onClose={closeDialog} />
      <ConfirmDialog confirm={confirm} onConfirm={handleConfirm} onClose={handleConfirmCancel} />
    </div>
  );
}
