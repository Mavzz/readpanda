import { useCallback, useState } from 'react';
import { Modal } from './ui';
import { ConfirmContext } from './useConfirm';

// Replaces window.confirm with the 12b modal. The text action confirms;
// ×, Escape and the backdrop cancel.
const ConfirmProvider = ({ children }) => {
  const [pending, setPending] = useState(null);

  const confirm = useCallback(
    (opts) => new Promise((resolve) => setPending({ ...opts, resolve })),
    []
  );

  const settle = useCallback((ok) => {
    setPending((p) => { p?.resolve(ok); return null; });
  }, []);
  const cancel = useCallback(() => settle(false), [settle]);

  return (
    <ConfirmContext.Provider value={confirm}>
      {children}
      {pending && (
        <Modal title={pending.title} onClose={cancel} action={{ label: pending.action ?? 'Delete', onClick: () => settle(true) }}>
          <p className="m-0 px-2 text-[13px] leading-normal text-ink-body whitespace-pre-line">{pending.message}</p>
        </Modal>
      )}
    </ConfirmContext.Provider>
  );
};

export default ConfirmProvider;
