import React, { useState } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { useGet } from '../services/useGet';
import { usePost } from '../services/usePost';
import { usePut } from '../services/usePut';
import { useDelete } from '../services/useDelete';
import { getBackendUrl } from '../utils/Helper';

const OurPicksPage = () => {
  const queryClient = useQueryClient();

  // Create form
  const [showCreateForm, setShowCreateForm] = useState(false);
  const [newTitle, setNewTitle] = useState('');
  const [newSortOrder, setNewSortOrder] = useState(0);
  const [newBookIds, setNewBookIds] = useState([]);
  const [newBookSearch, setNewBookSearch] = useState('');
  const [creating, setCreating] = useState(false);

  // Edit state: { id, title, sort_order, is_active }
  const [editing, setEditing] = useState(null);
  const [savingEdit, setSavingEdit] = useState(false);

  // Delete state
  const [deletingId, setDeletingId] = useState(null);

  // Book manager: which bucket's books panel is open
  const [managingBucketId, setManagingBucketId] = useState(null);
  const [bucketBooks, setBucketBooks] = useState([]);
  const [loadingBooks, setLoadingBooks] = useState(false);
  const [addingBookId, setAddingBookId] = useState(null);
  const [removingBookId, setRemovingBookId] = useState(null);
  const [bookSearch, setBookSearch] = useState('');

  const authHeaders = () => ({
    Authorization: `Bearer ${localStorage.getItem('token')}`,
  });

  const { data: buckets = [], isLoading: loading, error } = useQuery({
    queryKey: ['our-picks'],
    queryFn: async () => {
      const { response } = await useGet(await getBackendUrl('/home/our-picks'), authHeaders());
      return response.buckets || [];
    },
  });

  const { data: allBooks = [] } = useQuery({
    queryKey: ['books-all'],
    queryFn: async () => {
      const { response } = await useGet(await getBackendUrl('/books/all'), authHeaders());
      return response.books || [];
    },
    staleTime: Infinity,
  });

  // ── Create ──────────────────────────────────────────────────

  const handleCreate = async (e) => {
    e.preventDefault();
    const title = newTitle.trim();
    if (!title) return;
    setCreating(true);
    try {
      const { response } = await usePost(
        await getBackendUrl('/home/our-picks'),
        { title, sort_order: newSortOrder, book_ids: newBookIds },
        authHeaders()
      );
      const preview = allBooks
        .filter((b) => newBookIds.includes(b.id))
        .slice(0, 2)
        .map((b) => ({ book_id: b.id, title: b.title, cover_image_url: b.cover_image_url }));
      queryClient.setQueryData(['our-picks'], (prev) => [...(prev ?? []), { ...response, book_count: newBookIds.length, books_preview: preview }]);
      setNewTitle('');
      setNewSortOrder(0);
      setNewBookIds([]);
      setNewBookSearch('');
      setShowCreateForm(false);
    } catch (err) {
      alert(err.message);
    } finally {
      setCreating(false);
    }
  };

  // ── Edit ────────────────────────────────────────────────────

  const handleEditSave = async () => {
    if (!editing?.title?.trim()) return;
    setSavingEdit(true);
    try {
      const { response } = await usePut(
        await getBackendUrl(`/home/our-picks/${editing.id}`),
        { title: editing.title.trim(), sort_order: editing.sort_order, is_active: editing.is_active },
        authHeaders()
      );
      queryClient.setQueryData(['our-picks'], (prev) => (prev ?? []).map((b) => b.id === editing.id ? { ...b, ...response } : b));
      setEditing(null);
    } catch (err) {
      alert(err.message);
    } finally {
      setSavingEdit(false);
    }
  };

  // Toggle active without opening the full edit form
  const handleToggleActive = async (bucket) => {
    try {
      const { response } = await usePut(
        await getBackendUrl(`/home/our-picks/${bucket.id}`),
        { title: bucket.title, sort_order: bucket.sort_order, is_active: !bucket.is_active },
        authHeaders()
      );
      queryClient.setQueryData(['our-picks'], (prev) => (prev ?? []).map((b) => b.id === bucket.id ? { ...b, ...response } : b));
    } catch (err) {
      alert(err.message);
    }
  };

  // ── Delete ──────────────────────────────────────────────────

  const handleDelete = async (bucketId) => {
    if (!window.confirm('Delete this curated bucket? This cannot be undone.')) return;
    setDeletingId(bucketId);
    if (managingBucketId === bucketId) setManagingBucketId(null);
    try {
      await useDelete(await getBackendUrl(`/home/our-picks/${bucketId}`), authHeaders());
      queryClient.setQueryData(['our-picks'], (prev) => (prev ?? []).filter((b) => b.id !== bucketId));
    } catch (err) {
      alert(err.message);
    } finally {
      setDeletingId(null);
    }
  };

  // ── Manage books ────────────────────────────────────────────

  const openManageBooks = async (bucketId) => {
    setManagingBucketId(bucketId);
    setBookSearch('');
    setLoadingBooks(true);
    try {
      const { response } = await useGet(
        await getBackendUrl(`/home/our-picks/${bucketId}/books`),
        authHeaders()
      );
      setBucketBooks(response.books || []);
    } catch (err) {
      alert(err.message);
    } finally {
      setLoadingBooks(false);
    }
  };

  const handleAddBook = async (bookId) => {
    setAddingBookId(bookId);
    try {
      await usePost(
        await getBackendUrl(`/home/our-picks/${managingBucketId}/books`),
        { book_ids: [bookId] },
        authHeaders()
      );
      const book = allBooks.find((b) => b.id === bookId);
      if (book) setBucketBooks((prev) => [...prev, { book_id: book.id, title: book.title, cover_image_url: book.cover_image_url, genre: book.genre }]);
      queryClient.setQueryData(['our-picks'], (prev) => (prev ?? []).map((b) => b.id === managingBucketId ? { ...b, book_count: (b.book_count || 0) + 1 } : b));
    } catch (err) {
      alert(err.message);
    } finally {
      setAddingBookId(null);
    }
  };

  const handleRemoveBook = async (bookId) => {
    setRemovingBookId(bookId);
    try {
      await useDelete(
        await getBackendUrl(`/home/our-picks/${managingBucketId}/books/${bookId}`),
        authHeaders()
      );
      setBucketBooks((prev) => prev.filter((b) => b.book_id !== bookId));
      queryClient.setQueryData(['our-picks'], (prev) => (prev ?? []).map((b) => b.id === managingBucketId ? { ...b, book_count: Math.max(0, (b.book_count || 1) - 1) } : b));
    } catch (err) {
      alert(err.message);
    } finally {
      setRemovingBookId(null);
    }
  };

  const bucketBookIds = new Set(bucketBooks.map((b) => b.book_id));
  const filteredAllBooks = allBooks.filter((b) =>
    !bucketBookIds.has(b.id) &&
    (bookSearch === '' || b.title.toLowerCase().includes(bookSearch.toLowerCase()))
  );

  if (loading) return (
    <div className="flex items-center gap-3 text-gray-500 mt-10">
      <svg className="animate-spin h-5 w-5 text-indigo-500" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
        <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
        <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z" />
      </svg>
      Loading...
    </div>
  );
  if (error) return (
    <div className="flex items-center gap-3 bg-red-50 border border-red-200 text-red-700 rounded-lg px-4 py-3 mt-4 text-sm">
      <svg className="w-4 h-4 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
      </svg>
      {error.message}
    </div>
  );

  return (
    <div>
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-3xl font-bold text-gray-800">Our Picks</h1>
          <p className="text-sm text-gray-500 mt-1">Curated recommendation buckets visible to all users</p>
        </div>
        <button
          onClick={() => { setShowCreateForm(true); setNewTitle(''); setNewSortOrder(buckets.length); setNewBookIds([]); setNewBookSearch(''); }}
          className="inline-flex items-center gap-1.5 px-4 py-2 bg-indigo-600 text-white text-sm font-medium rounded-lg hover:bg-indigo-700 active:bg-indigo-800 transition-colors shadow-sm"
        >
          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
          </svg>
          New Pick
        </button>
      </div>

      {/* Create form */}
      {showCreateForm && (
        <div className="mb-6 bg-white rounded-xl shadow-md border border-gray-100 p-5">
          <h2 className="text-sm font-semibold text-gray-700 mb-4">New Curated Bucket</h2>
          <form onSubmit={handleCreate}>
            <div className="flex items-center gap-3 mb-4">
              <input
                type="text"
                value={newTitle}
                onChange={(e) => setNewTitle(e.target.value)}
                maxLength={100}
                placeholder="Title (e.g. Staff Picks, New Arrivals)"
                className="flex-1 border border-gray-300 rounded-lg px-3 py-2 text-sm text-gray-900 bg-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-transparent"
                autoFocus
              />
              <div className="flex flex-col items-center">
                <label className="text-[10px] text-gray-400 mb-0.5 font-medium uppercase tracking-wide">Order</label>
                <input
                  type="number"
                  value={newSortOrder}
                  onChange={(e) => setNewSortOrder(parseInt(e.target.value) || 0)}
                  className="w-16 border border-gray-300 rounded-lg px-2 py-2 text-sm text-gray-900 bg-white text-center focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-transparent"
                />
              </div>
            </div>

            {/* Book picker */}
            <div className="border border-gray-200 rounded-lg overflow-hidden mb-4">
              <div className="flex items-center justify-between px-3 py-2 bg-gray-50 border-b border-gray-200">
                <div className="flex items-center gap-2">
                  <span className="text-xs font-semibold text-gray-600">Add Books</span>
                  {newBookIds.length > 0 && (
                    <span className="inline-flex items-center px-1.5 py-0.5 rounded-full text-[10px] font-semibold bg-indigo-100 text-indigo-700">
                      {newBookIds.length} selected
                    </span>
                  )}
                </div>
                <input
                  type="text"
                  value={newBookSearch}
                  onChange={(e) => setNewBookSearch(e.target.value)}
                  placeholder="Search books..."
                  className="border border-gray-300 rounded-md px-2 py-1 text-xs text-gray-900 bg-white placeholder-gray-400 focus:outline-none focus:ring-1 focus:ring-indigo-500 w-44"
                />
              </div>
              <div className="max-h-48 overflow-y-auto">
                {allBooks
                  .filter((b) => newBookSearch === '' || b.title.toLowerCase().includes(newBookSearch.toLowerCase()))
                  .map((book) => {
                    const selected = newBookIds.includes(book.id);
                    return (
                      <div
                        key={book.id}
                        className={`flex items-center gap-3 py-2 px-3 cursor-pointer transition-colors border-b border-gray-100 last:border-0 ${selected ? 'bg-indigo-50' : 'hover:bg-gray-50'}`}
                        onClick={() =>
                          setNewBookIds((prev) =>
                            selected ? prev.filter((id) => id !== book.id) : [...prev, book.id]
                          )
                        }
                      >
                        <input
                          type="checkbox"
                          readOnly
                          checked={selected}
                          className="accent-indigo-600 shrink-0 w-4 h-4 pointer-events-none"
                        />
                        {book.cover_image_url ? (
                          <img src={book.cover_image_url} alt="" className="w-7 h-10 object-cover rounded shadow-sm shrink-0" />
                        ) : (
                          <div className="w-7 h-10 bg-gray-100 rounded shrink-0" />
                        )}
                        <span className="text-xs text-gray-800 line-clamp-1 flex-1">{book.title}</span>
                      </div>
                    );
                  })}
                {allBooks.length === 0 && (
                  <p className="text-xs text-gray-400 py-4 text-center">No books available</p>
                )}
              </div>
            </div>

            <div className="flex gap-2">
              <button
                type="submit"
                disabled={creating || !newTitle.trim()}
                className="px-4 py-2 bg-indigo-600 text-white text-sm font-medium rounded-lg hover:bg-indigo-700 disabled:opacity-50 transition-colors shadow-sm"
              >
                {creating ? 'Creating...' : `Create${newBookIds.length > 0 ? ` with ${newBookIds.length} book${newBookIds.length !== 1 ? 's' : ''}` : ''}`}
              </button>
              <button
                type="button"
                onClick={() => setShowCreateForm(false)}
                className="px-4 py-2 bg-white text-gray-700 text-sm font-medium rounded-lg border border-gray-300 hover:bg-gray-50 transition-colors"
              >
                Cancel
              </button>
            </div>
          </form>
        </div>
      )}

      {/* Bucket list */}
      {buckets.length === 0 ? (
        <div className="bg-white rounded-xl border-2 border-dashed border-gray-200 p-12 text-center">
          <div className="w-12 h-12 rounded-full bg-indigo-50 flex items-center justify-center mx-auto mb-3">
            <svg className="w-6 h-6 text-indigo-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
            </svg>
          </div>
          <p className="text-gray-700 font-medium">No curated buckets yet</p>
          <p className="text-sm text-gray-400 mt-1">Click &ldquo;New Pick&rdquo; to create your first one.</p>
        </div>
      ) : (
        <div className="space-y-3">
          {buckets.map((bucket) => (
            <div
              key={bucket.id}
              className={`bg-white rounded-xl shadow-sm border overflow-hidden transition-colors ${
                bucket.is_active ? 'border-l-4 border-l-green-400 border-gray-100' : 'border-gray-200'
              }`}
            >
              {/* Bucket row */}
              <div className="flex items-center gap-4 px-5 py-3.5">
                {/* Sort order badge */}
                <span className="flex-shrink-0 w-7 h-7 rounded-full bg-indigo-100 text-indigo-700 text-xs font-bold flex items-center justify-center">
                  {bucket.sort_order}
                </span>

                {/* Title / edit */}
                <div className="flex-1 min-w-0">
                  {editing?.id === bucket.id ? (
                    <div className="flex items-center gap-2">
                      <input
                        type="text"
                        value={editing.title}
                        onChange={(e) => setEditing({ ...editing, title: e.target.value })}
                        maxLength={100}
                        className="flex-1 border border-gray-300 rounded-lg px-2 py-1 text-sm text-gray-900 bg-white focus:outline-none focus:ring-2 focus:ring-indigo-500"
                        autoFocus
                        onKeyDown={(e) => { if (e.key === 'Enter') handleEditSave(); if (e.key === 'Escape') setEditing(null); }}
                      />
                      <input
                        type="number"
                        value={editing.sort_order}
                        onChange={(e) => setEditing({ ...editing, sort_order: parseInt(e.target.value) || 0 })}
                        className="w-14 border border-gray-300 rounded-lg px-2 py-1 text-sm text-gray-900 bg-white text-center focus:outline-none focus:ring-2 focus:ring-indigo-500"
                      />
                      <button onClick={handleEditSave} disabled={savingEdit} className="px-2.5 py-1 bg-indigo-600 text-white text-xs font-medium rounded-md hover:bg-indigo-700 disabled:opacity-50">Save</button>
                      <button onClick={() => setEditing(null)} className="px-2.5 py-1 bg-gray-100 text-gray-600 text-xs font-medium rounded-md hover:bg-gray-200">Cancel</button>
                    </div>
                  ) : (
                    <div className="flex items-center gap-2.5 min-w-0">
                      <span className="font-semibold text-gray-800 truncate">{bucket.title}</span>
                      <span className="flex-shrink-0 text-xs text-gray-400 tabular-nums">{bucket.book_count ?? 0} books</span>
                      {bucket.books_preview?.length > 0 && (
                        <div className="flex -space-x-1.5 flex-shrink-0">
                          {bucket.books_preview.slice(0, 4).map((b) => (
                            <img
                              key={b.book_id}
                              src={b.cover_image_url}
                              alt={b.title}
                              className="w-5 h-7 object-cover rounded shadow-sm ring-1 ring-white"
                            />
                          ))}
                        </div>
                      )}
                    </div>
                  )}
                </div>

                {/* Active toggle */}
                <button
                  onClick={() => handleToggleActive(bucket)}
                  title={bucket.is_active ? 'Active — click to deactivate' : 'Inactive — click to activate'}
                  className={`relative flex-shrink-0 inline-flex h-5 w-9 items-center rounded-full transition-colors focus:outline-none ${
                    bucket.is_active ? 'bg-green-400' : 'bg-gray-300'
                  }`}
                >
                  <span className={`inline-block h-3.5 w-3.5 transform rounded-full bg-white shadow transition-transform ${
                    bucket.is_active ? 'translate-x-[19px]' : 'translate-x-[2px]'
                  }`} />
                </button>

                {/* Actions */}
                <div className="flex-shrink-0 flex items-center gap-1">
                  <button
                    onClick={() => managingBucketId === bucket.id ? setManagingBucketId(null) : openManageBooks(bucket.id)}
                    className="px-2.5 py-1.5 rounded-md text-xs font-medium text-indigo-600 hover:bg-indigo-50 transition-colors"
                  >
                    {managingBucketId === bucket.id ? 'Hide Books' : 'Manage Books'}
                  </button>
                  <button
                    onClick={() => setEditing({ id: bucket.id, title: bucket.title, sort_order: bucket.sort_order, is_active: bucket.is_active })}
                    className="px-2.5 py-1.5 rounded-md text-xs font-medium text-gray-600 hover:bg-gray-100 transition-colors"
                  >
                    Edit
                  </button>
                  <button
                    onClick={() => handleDelete(bucket.id)}
                    disabled={deletingId === bucket.id}
                    className="px-2.5 py-1.5 rounded-md text-xs font-medium text-red-500 hover:bg-red-50 transition-colors disabled:opacity-50"
                  >
                    {deletingId === bucket.id ? 'Deleting…' : 'Delete'}
                  </button>
                </div>
              </div>

              {/* Book manager panel */}
              {managingBucketId === bucket.id && (
                <div className="border-t border-gray-100 bg-gray-50 px-5 py-4">
                  <div className="flex gap-6">
                    {/* Current books */}
                    <div className="flex-1 min-w-0">
                      <h4 className="text-xs font-semibold text-gray-500 uppercase tracking-wide mb-2">In this bucket</h4>
                      {loadingBooks ? (
                        <div className="flex items-center gap-2 text-sm text-gray-400 py-2">
                          <svg className="animate-spin h-4 w-4 text-indigo-400" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                            <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                            <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z" />
                          </svg>
                          Loading…
                        </div>
                      ) : bucketBooks.length === 0 ? (
                        <p className="text-sm text-gray-400 py-2">No books yet. Add from the right.</p>
                      ) : (
                        <ul className="space-y-1.5 max-h-60 overflow-y-auto">
                          {bucketBooks.map((b) => (
                            <li key={b.book_id} className="flex items-center gap-3 bg-white rounded-lg px-3 py-2 shadow-sm border border-gray-100">
                              <img
                                src={b.cover_image_url || `https://placehold.co/40x56/6366F1/FFFFFF?text=...`}
                                alt={b.title}
                                className="w-7 h-10 object-cover rounded shadow-sm"
                              />
                              <span className="flex-1 text-sm text-gray-800 truncate">{b.title}</span>
                              <button
                                onClick={() => handleRemoveBook(b.book_id)}
                                disabled={removingBookId === b.book_id}
                                className="flex-shrink-0 w-6 h-6 rounded-full flex items-center justify-center text-gray-400 hover:text-red-500 hover:bg-red-50 transition-colors disabled:opacity-40 text-sm"
                                title="Remove"
                              >
                                ✕
                              </button>
                            </li>
                          ))}
                        </ul>
                      )}
                    </div>

                    {/* Divider */}
                    <div className="w-px bg-gray-200" />

                    {/* Add books */}
                    <div className="flex-1 min-w-0">
                      <h4 className="text-xs font-semibold text-gray-500 uppercase tracking-wide mb-2">Add books</h4>
                      <input
                        type="text"
                        value={bookSearch}
                        onChange={(e) => setBookSearch(e.target.value)}
                        placeholder="Search books..."
                        className="w-full border border-gray-300 rounded-lg px-3 py-1.5 text-sm text-gray-900 bg-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-indigo-500 mb-2"
                      />
                      {filteredAllBooks.length === 0 ? (
                        <p className="text-sm text-gray-400 py-2">All books are already in this bucket.</p>
                      ) : (
                        <ul className="space-y-1.5 max-h-60 overflow-y-auto">
                          {filteredAllBooks.map((b) => (
                            <li key={b.id} className="flex items-center gap-3 bg-white rounded-lg px-3 py-2 shadow-sm border border-gray-100">
                              <img
                                src={b.cover_image_url || `https://placehold.co/40x56/6366F1/FFFFFF?text=...`}
                                alt={b.title}
                                className="w-7 h-10 object-cover rounded shadow-sm"
                              />
                              <span className="flex-1 text-sm text-gray-800 truncate">{b.title}</span>
                              <button
                                onClick={() => handleAddBook(b.book_id)}
                                disabled={addingBookId === b.book_id}
                                className="flex-shrink-0 px-2.5 py-1 rounded-full bg-indigo-50 text-indigo-600 hover:bg-indigo-100 text-xs font-semibold transition-colors disabled:opacity-50"
                              >
                                {addingBookId === b.book_id ? '…' : '+ Add'}
                              </button>
                            </li>
                          ))}
                        </ul>
                      )}
                    </div>
                  </div>
                </div>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
};

export default OurPicksPage;
