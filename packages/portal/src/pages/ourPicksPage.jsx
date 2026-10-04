import { useEffect, useState } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { useSearchParams } from 'react-router-dom';
import {
  Banner, BookRow, Button, Card, CoverStack, CreateSlot, Eyebrow, Field, IconButton, Loading, Modal, PageHeader, SearchInput, TextAction, Toggle,
} from '../components/ui';
import { CloseIcon } from '../components/icons';
import { useConfirm } from '../components/useConfirm';
import { api } from '../services/api';
import { OUR_PICKS_KEY, bookMeta, useAllBooks, useOurPicks } from '../services/queries';

const bucketBooksKey = (id) => ['our-picks-books', id];

const matches = (book, q) => !q || book.title.toLowerCase().includes(q) || (book.author_name ?? '').toLowerCase().includes(q);

const plural = (n, word) => `${n} ${word}${n === 1 ? '' : 's'}`;

// ── Create / edit modal ───────────────────────────────────────

const BucketModal = ({ bucket, nextOrder, allBooks, onClose, onSaved }) => {
  const editing = !!bucket;
  const [title, setTitle] = useState(bucket?.title ?? '');
  const [order, setOrder] = useState(bucket?.sort_order ?? nextOrder);
  const [active, setActive] = useState(bucket?.is_active ?? true);
  const [picked, setPicked] = useState([]);
  const [search, setSearch] = useState('');
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');

  const valid = title.trim().length > 0 && !saving;
  const q = search.trim().toLowerCase();

  const save = async () => {
    if (!valid) return;
    setSaving(true);
    setError('');
    try {
      const body = { title: title.trim(), sort_order: Number(order) || 0 };
      if (editing) {
        const res = await api.put(`/home/our-picks/${bucket.id}`, { ...body, is_active: active });
        onSaved({ ...bucket, ...res });
      } else {
        const res = await api.post('/home/our-picks', { ...body, book_ids: picked });
        const preview = picked
          .map((id) => allBooks.find((b) => b.id === id))
          .filter(Boolean)
          .map((b) => ({ book_id: b.id, title: b.title, cover_image_url: b.cover_image_url }));
        onSaved({ ...res, books_preview: preview });
      }
      onClose();
    } catch (err) {
      setError(`Couldn't save: ${err.message}`);
      setSaving(false);
    }
  };

  return (
    <Modal
      title={editing ? 'Edit bucket' : 'New curated bucket'}
      onClose={onClose}
      width={editing ? 480 : 560}
      action={{ label: saving ? 'Saving…' : editing ? 'Save' : 'Create', onClick: save, disabled: !valid }}
    >
      <Banner message={error} />
      <form onSubmit={(e) => { e.preventDefault(); save(); }} className="flex flex-col gap-4">
        <div className="grid grid-cols-[1fr_96px] gap-3">
          <Field label="Title" value={title} max={100} onChange={(e) => setTitle(e.target.value)} placeholder="Quiet English novels" autoFocus />
          <Field label="Order" type="number" value={order} onChange={(e) => setOrder(e.target.value)} hint="Lower first" />
        </div>
        {editing && (
          <div className="flex items-center gap-3">
            <Toggle on={active} label="Featured on Discover" onChange={setActive} />
            <span className="text-xs font-semibold text-ink-body">Featured on Discover</span>
          </div>
        )}
        <button type="submit" hidden />
      </form>

      {!editing && (
        <div className="flex flex-col gap-2.5">
          <div className="flex items-center justify-between">
            <Eyebrow>Books{picked.length > 0 && ` · ${picked.length} picked`}</Eyebrow>
          </div>
          <SearchInput value={search} onChange={setSearch} placeholder="Search books by title or author" />
          <div className="max-h-72 overflow-y-auto flex flex-col">
            {allBooks.filter((b) => matches(b, q)).map((book) => {
              const on = picked.includes(book.id);
              return (
                <button
                  key={book.id}
                  type="button"
                  aria-pressed={on}
                  onClick={() => setPicked((p) => (on ? p.filter((id) => id !== book.id) : [...p, book.id]))}
                  className={`text-left px-3 py-2 rounded-field ${on ? 'bg-surface-2' : 'bg-transparent hover:bg-surface-1'}`}
                >
                  <BookRow
                    cover={book.cover_image_url}
                    title={book.title}
                    meta={bookMeta(book) || book.genre}
                    trailing={
                      <span className={`w-5 h-5 rounded-full shrink-0 flex items-center justify-center text-[11px] font-extrabold ${on ? 'bg-gold text-on-gold' : 'shadow-[inset_0_0_0_1.5px_var(--rp-surface-3)]'}`}>
                        {on ? picked.indexOf(book.id) + 1 : ''}
                      </span>
                    }
                  />
                </button>
              );
            })}
            {allBooks.length === 0 && <p className="m-0 py-4 text-center text-[13px] text-ink-holder">No books uploaded yet.</p>}
          </div>
        </div>
      )}
    </Modal>
  );
};

// ── Books in one bucket ───────────────────────────────────────

const BucketBooksPanel = ({ bucket, allBooks, onError }) => {
  const queryClient = useQueryClient();
  const [search, setSearch] = useState('');
  const [busyId, setBusyId] = useState(null);

  const { data: inBucket = [], isLoading, error, refetch } = useQuery({
    queryKey: bucketBooksKey(bucket.id),
    queryFn: async () => (await api.get(`/home/our-picks/${bucket.id}/books`)).books ?? [],
  });

  const patchBucket = (fn) =>
    queryClient.setQueryData(OUR_PICKS_KEY, (prev) => (prev ?? []).map((b) => (b.id === bucket.id ? fn(b) : b)));

  // /books/all keys books by `id`; bucket entries by `book_id`.
  const add = async (book) => {
    setBusyId(book.id);
    try {
      await api.post(`/home/our-picks/${bucket.id}/books`, { book_ids: [book.id] });
      const entry = { book_id: book.id, title: book.title, cover_image_url: book.cover_image_url, author_name: book.author_name };
      queryClient.setQueryData(bucketBooksKey(bucket.id), (prev) => [...(prev ?? []), entry]);
      patchBucket((b) => ({ ...b, book_count: (b.book_count ?? 0) + 1, books_preview: [...(b.books_preview ?? []), entry] }));
    } catch (err) {
      onError(`Couldn't add "${book.title}": ${err.message}`);
    } finally {
      setBusyId(null);
    }
  };

  const remove = async (book) => {
    setBusyId(book.book_id);
    try {
      await api.del(`/home/our-picks/${bucket.id}/books/${book.book_id}`);
      queryClient.setQueryData(bucketBooksKey(bucket.id), (prev) => (prev ?? []).filter((b) => b.book_id !== book.book_id));
      patchBucket((b) => ({
        ...b,
        book_count: Math.max(0, (b.book_count ?? 1) - 1),
        books_preview: (b.books_preview ?? []).filter((p) => p.book_id !== book.book_id),
      }));
    } catch (err) {
      onError(`Couldn't remove "${book.title}": ${err.message}`);
    } finally {
      setBusyId(null);
    }
  };

  const inIds = new Set(inBucket.map((b) => b.book_id));
  const q = search.trim().toLowerCase();
  const addable = allBooks.filter((b) => !inIds.has(b.id) && matches(b, q));

  return (
    <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 pt-4 mt-4 shadow-[inset_0_1px_0_var(--rp-hairline)]">
      <div className="flex flex-col gap-2.5 min-w-0">
        <Eyebrow>In this bucket · {inBucket.length}</Eyebrow>
        {error ? (
          <Banner message="Couldn't load this bucket's books." onRetry={refetch} />
        ) : isLoading ? (
          <Loading />
        ) : inBucket.length === 0 ? (
          <p className="m-0 text-[13px] text-ink-holder">No books yet. Add some from the list.</p>
        ) : (
          <div className="max-h-80 overflow-y-auto flex flex-col">
            {inBucket.map((b) => (
              <div key={b.book_id} className="py-2 hairline-b last:shadow-none">
                <BookRow
                  cover={b.cover_image_url}
                  title={b.title}
                  meta={b.author_name}
                  trailing={
                    <IconButton label={`Remove ${b.title}`} disabled={busyId === b.book_id} onClick={() => remove(b)}>
                      <CloseIcon width={14} height={14} strokeWidth={2.5} />
                    </IconButton>
                  }
                />
              </div>
            ))}
          </div>
        )}
      </div>

      <div className="flex flex-col gap-2.5 min-w-0">
        <Eyebrow>Add books</Eyebrow>
        <SearchInput value={search} onChange={setSearch} placeholder="Search books by title or author" />
        {addable.length === 0 ? (
          <p className="m-0 text-[13px] text-ink-holder">{q ? 'Nothing matches that search.' : 'Every book is already in this bucket.'}</p>
        ) : (
          <div className="max-h-80 overflow-y-auto flex flex-col">
            {addable.map((b) => (
              <div key={b.id} className="py-2 hairline-b last:shadow-none">
                <BookRow
                  cover={b.cover_image_url}
                  title={b.title}
                  meta={bookMeta(b) || b.genre}
                  trailing={
                    <TextAction disabled={busyId === b.id} onClick={() => add(b)}>
                      {busyId === b.id ? 'Adding…' : 'Add'}
                    </TextAction>
                  }
                />
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
};

// ── Page ──────────────────────────────────────────────────────

const OurPicksPage = () => {
  const queryClient = useQueryClient();
  const confirm = useConfirm();
  const [params, setParams] = useSearchParams();

  const { data: buckets = [], isLoading, error, refetch } = useOurPicks();
  const { data: allBooks = [] } = useAllBooks();

  // { bucket } to edit, {} to create. The dashboard's create slot links here with ?new=1.
  const [modal, setModal] = useState(() => (params.get('new') ? {} : null));
  const [managingId, setManagingId] = useState(null);
  const [togglingId, setTogglingId] = useState(null);
  const [banner, setBanner] = useState('');

  useEffect(() => {
    if (params.get('new')) setParams({}, { replace: true });
  }, [params, setParams]);

  const sorted = [...buckets].sort((a, b) => a.sort_order - b.sort_order);
  const nextOrder = buckets.reduce((m, b) => Math.max(m, b.sort_order + 1), 0);

  const setBuckets = (fn) => queryClient.setQueryData(OUR_PICKS_KEY, (prev) => fn(prev ?? []));

  const onSaved = (saved) =>
    setBuckets((prev) => (prev.some((b) => b.id === saved.id) ? prev.map((b) => (b.id === saved.id ? saved : b)) : [...prev, saved]));

  const toggleActive = async (bucket) => {
    setTogglingId(bucket.id);
    try {
      const res = await api.put(`/home/our-picks/${bucket.id}`, { title: bucket.title, sort_order: bucket.sort_order, is_active: !bucket.is_active });
      onSaved({ ...bucket, ...res });
    } catch (err) {
      setBanner(`Couldn't update "${bucket.title}": ${err.message}`);
    } finally {
      setTogglingId(null);
    }
  };

  const remove = async (bucket) => {
    const ok = await confirm({
      title: 'Delete bucket',
      message: `"${bucket.title}" comes off Discover for every reader. The books themselves stay.`,
    });
    if (!ok) return;
    try {
      await api.del(`/home/our-picks/${bucket.id}`);
      if (managingId === bucket.id) setManagingId(null);
      setBuckets((prev) => prev.filter((b) => b.id !== bucket.id));
    } catch (err) {
      setBanner(`Couldn't delete "${bucket.title}": ${err.message}`);
    }
  };

  return (
    <>
      <PageHeader title="Our picks" subtitle="Curated buckets readers see on Discover">
        <Button onClick={() => setModal({})}>New curated bucket</Button>
      </PageHeader>

      <Banner message={banner} onDismiss={() => setBanner('')} />

      {error ? (
        <Banner message="Couldn't load curated buckets." onRetry={refetch} />
      ) : isLoading ? (
        <Loading />
      ) : (
        <div className="flex flex-col gap-3">
          {sorted.map((bucket) => {
            const open = managingId === bucket.id;
            return (
              <Card key={bucket.id} className="px-[18px] py-4">
                <div className="flex flex-wrap items-center gap-4">
                  <CoverStack books={bucket.books_preview} />
                  <div className="flex-1 min-w-[180px] flex flex-col gap-1">
                    <Eyebrow className={bucket.is_active ? 'text-gold!' : ''}>{bucket.is_active ? 'Featured' : 'Hidden'} · #{bucket.sort_order}</Eyebrow>
                    <span className="text-sm leading-[1.3] font-extrabold text-ink-title truncate">{bucket.title}</span>
                    <span className="text-xs font-semibold text-ink-meta">{plural(bucket.book_count ?? 0, 'book')}</span>
                  </div>
                  <Toggle
                    on={bucket.is_active}
                    label={`Feature "${bucket.title}" on Discover`}
                    disabled={togglingId === bucket.id}
                    onChange={() => toggleActive(bucket)}
                  />
                  <div className="flex items-center gap-4">
                    <TextAction onClick={() => setManagingId(open ? null : bucket.id)}>{open ? 'Hide books' : 'Books'}</TextAction>
                    <TextAction tone="muted" onClick={() => setModal({ bucket })}>Edit</TextAction>
                    <TextAction tone="muted" onClick={() => remove(bucket)}>Delete</TextAction>
                  </div>
                </div>
                {open && <BucketBooksPanel bucket={bucket} allBooks={allBooks} onError={setBanner} />}
              </Card>
            );
          })}
          <CreateSlot label={sorted.length ? 'New curated bucket' : 'Create the first curated bucket'} onClick={() => setModal({})} />
        </div>
      )}

      {modal && (
        <BucketModal
          bucket={modal.bucket}
          nextOrder={nextOrder}
          allBooks={allBooks}
          onClose={() => setModal(null)}
          onSaved={onSaved}
        />
      )}
    </>
  );
};

export default OurPicksPage;
