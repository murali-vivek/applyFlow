export default function Dialog({
  open,
  title,
  message,
  variant = 'success',
  confirmLabel = 'Confirm',
  cancelLabel = 'Cancel',
  onConfirm,
  onClose,
}) {
  if (!open) return null;

  const isConfirm = variant === 'confirm';

  return (
    <div className="dialog-overlay" onClick={onClose} role="presentation">
      <div
        className={`dialog dialog-${isConfirm ? 'confirm' : variant}`}
        role="dialog"
        aria-modal="true"
        aria-labelledby="dialog-title"
        onClick={(e) => e.stopPropagation()}
      >
        <h3 id="dialog-title">{title}</h3>
        <p className="dialog-message">{message}</p>
        <div className="dialog-actions">
          {isConfirm ? (
            <>
              <button type="button" className="btn btn-secondary" onClick={onClose}>
                {cancelLabel}
              </button>
              <button type="button" className="btn btn-primary" onClick={onConfirm}>
                {confirmLabel}
              </button>
            </>
          ) : (
            <button type="button" className="btn btn-primary" onClick={onClose}>
              Close
            </button>
          )}
        </div>
      </div>
    </div>
  );
}
