import { useState } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { Field, Modal } from './ui';
import { MODERATION_KEYS, NOTE_MAX, resolveReport, setUserActive } from '../services/moderation';

// Suspending and reinstating, with an optional note for the history
// (portal-moderation R3.2). The confirm modal has no input, so this is its
// own dialog. It makes the call itself, so a failure keeps the dialog and
// the note on screen.
const SuspendDialog = ({ request, onDone }) => {
  const { username, suspending, reportId, uuid, message } = request;
  const queryClient = useQueryClient();
  const [note, setNote] = useState('');
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');
  const title = suspending ? `Suspend ${username}` : `Reinstate ${username}`;

  const submit = async () => {
    setSaving(true);
    setError('');
    try {
      if (reportId) await resolveReport(reportId, 'suspend_user', note);
      else await setUserActive(uuid, !suspending, note);
      MODERATION_KEYS.forEach((queryKey) => queryClient.invalidateQueries({ queryKey }));
      onDone(true);
    } catch (err) {
      setError(`Couldn't do that: ${err.message}`);
      setSaving(false);
    }
  };

  return (
    <Modal
      title={title}
      onClose={() => { if (!saving) onDone(false); }}
      action={{ label: suspending ? 'Suspend' : 'Reinstate', onClick: submit, disabled: saving }}
    >
      <p className="m-0 px-2 text-[13px] leading-normal text-ink-body">
        {message ?? (suspending
          ? `${username} is signed out and can't sign in or post until you reinstate them. Their comments stay.`
          : `${username} can sign in and post again.`)}
      </p>
      <Field
        label="Note (optional)"
        multiline
        max={NOTE_MAX}
        value={note}
        onChange={(e) => setNote(e.target.value)}
        placeholder={suspending ? 'Why, for the next admin' : 'Why they\'re back'}
      />
      <p className="m-0 px-2 text-[11px] font-semibold text-ink-holder">Admins only. The reader never sees this.</p>
      {error && <p role="alert" className="m-0 px-2 text-[13px] text-ink-body">{error}</p>}
    </Modal>
  );
};

export default SuspendDialog;
