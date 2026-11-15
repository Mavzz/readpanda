import { useCallback, useState } from 'react';
import SuspendDialog from './SuspendDialog';

// const { suspendReader, dialog } = useSuspendReader();
// const done = await suspendReader({ username, suspending, uuid } | { username, suspending: true, reportId });
// Render {dialog} somewhere in the page. Resolves true once it's done,
// false if the admin backs out.
export const useSuspendReader = () => {
  const [request, setRequest] = useState(null);
  const suspendReader = useCallback(
    (opts) => new Promise((resolve) => setRequest({ ...opts, resolve })),
    [],
  );
  const onDone = useCallback((ok) => {
    setRequest((r) => { r?.resolve(ok); return null; });
  }, []);
  const dialog = request ? <SuspendDialog key={request.uuid ?? request.reportId} request={request} onDone={onDone} /> : null;
  return { suspendReader, dialog };
};
