import Dialog from './Dialog';

export default function ConfirmDialog({ confirm, onConfirm, onClose }) {
  return (
    <Dialog
      open={confirm.open}
      title={confirm.title}
      message={confirm.message}
      variant="confirm"
      confirmLabel={confirm.confirmLabel}
      onConfirm={onConfirm}
      onClose={onClose}
    />
  );
}
