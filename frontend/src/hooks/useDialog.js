import { useState, useCallback, useRef } from 'react';

export function useDialog() {
  const [dialog, setDialog] = useState({
    open: false,
    title: '',
    message: '',
    variant: 'error',
  });

  const confirmResolver = useRef(null);
  const [confirm, setConfirm] = useState({
    open: false,
    title: '',
    message: '',
    confirmLabel: 'Confirm',
  });

  const showError = useCallback((message, title = 'Error') => {
    setDialog({ open: true, title, message, variant: 'error' });
  }, []);

  const showSuccess = useCallback((message, title = 'Success') => {
    setDialog({ open: true, title, message, variant: 'success' });
  }, []);

  const closeDialog = useCallback(() => {
    setDialog((d) => ({ ...d, open: false }));
  }, []);

  const showConfirm = useCallback((message, title = 'Confirm', confirmLabel = 'Confirm') => {
    return new Promise((resolve) => {
      confirmResolver.current = resolve;
      setConfirm({ open: true, title, message, confirmLabel });
    });
  }, []);

  const handleConfirm = useCallback(() => {
    confirmResolver.current?.(true);
    confirmResolver.current = null;
    setConfirm((c) => ({ ...c, open: false }));
  }, []);

  const handleConfirmCancel = useCallback(() => {
    confirmResolver.current?.(false);
    confirmResolver.current = null;
    setConfirm((c) => ({ ...c, open: false }));
  }, []);

  return {
    dialog,
    confirm,
    showError,
    showSuccess,
    showConfirm,
    closeDialog,
    handleConfirm,
    handleConfirmCancel,
  };
}
