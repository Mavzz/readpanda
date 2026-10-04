import { useState } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { Banner, Button, Card, CoverStack, CreateSlot, Field, Loading, Modal, PageHeader, TextAction } from '../components/ui';
import { useConfirm } from '../components/useConfirm';
import { api } from '../services/api';

const BUCKETS_KEY = ['my-buckets'];
const NAME_MAX = 40;

// New bucket / rename: same modal, the text action turns gold once the name is valid.
const NameModal = ({ bucket, onClose, onSaved }) => {
  const [name, setName] = useState(bucket?.name ?? '');
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');
  const trimmed = name.trim();
  const valid = trimmed.length > 0 && trimmed !== bucket?.name && !saving;

  const save = async () => {
    if (!valid) return;
    setSaving(true);
    setError('');
    try {
      if (bucket) {
        await api.put(`/users/me/buckets/${bucket.id}`, { name: trimmed });
        onSaved({ ...bucket, name: trimmed });
      } else {
        const res = await api.post('/users/me/buckets', { name: trimmed });
        onSaved({ ...res, books_preview: [] });
      }
      onClose();
    } catch (err) {
      setError(`Couldn't save: ${err.message}`);
      setSaving(false);
    }
  };

  return (
    <Modal
      title={bucket ? 'Rename bucket' : 'New bucket'}
      onClose={onClose}
      action={{ label: saving ? 'Saving…' : bucket ? 'Save' : 'Create', onClick: save, disabled: !valid }}
    >
      <Banner message={error} />
      <form onSubmit={(e) => { e.preventDefault(); save(); }}>
        <Field label="Bucket name" value={name} max={NAME_MAX} onChange={(e) => setName(e.target.value)} placeholder="Winter reads" autoFocus />
      </form>
    </Modal>
  );
};

const MyBucketsPage = () => {
  const queryClient = useQueryClient();
  const confirm = useConfirm();
  const [modal, setModal] = useState(null);
  const [banner, setBanner] = useState('');

  const { data: buckets = [], isLoading, error, refetch } = useQuery({
    queryKey: BUCKETS_KEY,
    queryFn: async () => (await api.get('/users/me/buckets')).buckets ?? [],
  });

  const setBuckets = (fn) => queryClient.setQueryData(BUCKETS_KEY, (prev) => fn(prev ?? []));

  const onSaved = (saved) =>
    setBuckets((prev) => (prev.some((b) => b.id === saved.id) ? prev.map((b) => (b.id === saved.id ? saved : b)) : [...prev, saved]));

  const remove = async (bucket) => {
    const ok = await confirm({ title: 'Delete bucket', message: `"${bucket.name}" goes away. The books in it stay in the library.` });
    if (!ok) return;
    try {
      await api.del(`/users/me/buckets/${bucket.id}`);
      setBuckets((prev) => prev.filter((b) => b.id !== bucket.id));
    } catch (err) {
      setBanner(`Couldn't delete "${bucket.name}": ${err.message}`);
    }
  };

  return (
    <>
      <PageHeader title="Buckets" subtitle="Your own reading buckets">
        <Button onClick={() => setModal({})}>New bucket</Button>
      </PageHeader>

      <Banner message={banner} onDismiss={() => setBanner('')} />

      {error ? (
        <Banner message="Couldn't load your buckets." onRetry={refetch} />
      ) : isLoading ? (
        <Loading />
      ) : (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
          {buckets.map((bucket) => (
            <Card key={bucket.id} className="px-[18px] py-4 flex items-center gap-4">
              <CoverStack books={bucket.books_preview} />
              <div className="flex-1 min-w-0 flex flex-col gap-1">
                <span className="text-sm leading-[1.3] font-extrabold text-ink-title truncate">{bucket.name}</span>
                <span className="text-xs font-semibold text-ink-meta">
                  {bucket.book_count ?? 0} {bucket.book_count === 1 ? 'book' : 'books'}
                </span>
                <div className="flex gap-4 mt-1">
                  <TextAction onClick={() => setModal({ bucket })}>Rename</TextAction>
                  <TextAction tone="muted" onClick={() => remove(bucket)}>Delete</TextAction>
                </div>
              </div>
            </Card>
          ))}
          <CreateSlot label={buckets.length ? 'New bucket' : 'Create your first bucket'} onClick={() => setModal({})} />
        </div>
      )}

      {modal && <NameModal bucket={modal.bucket} onClose={() => setModal(null)} onSaved={onSaved} />}
    </>
  );
};

export default MyBucketsPage;
