import React, { useState, useEffect } from 'react';
import { useGet } from '../services/useGet';
import { usePost } from '../services/usePost';
import { usePut } from '../services/usePut';
import { useDelete } from '../services/useDelete';
import { getBackendUrl } from '../utils/Helper';

const MyBucketsPage = () => {
  const [buckets, setBuckets] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  // Create bucket state
  const [showCreateForm, setShowCreateForm] = useState(false);
  const [newBucketName, setNewBucketName] = useState('');
  const [creating, setCreating] = useState(false);

  // Rename state: { bucketId, name }
  const [renaming, setRenaming] = useState(null);
  const [savingRename, setSavingRename] = useState(false);

  // Delete state
  const [deletingId, setDeletingId] = useState(null);

  const authHeaders = () => ({
    Authorization: `Bearer ${localStorage.getItem('token')}`,
  });

  const fetchBuckets = async () => {
    setLoading(true);
    setError(null);
    try {
      const { response } = await useGet(await getBackendUrl('/users/me/buckets'), authHeaders());
      setBuckets(response.buckets || []);
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchBuckets();
  }, []);

  const handleCreate = async (e) => {
    e.preventDefault();
    const name = newBucketName.trim();
    if (!name) return;
    setCreating(true);
    try {
      const { response } = await usePost(
        await getBackendUrl('/users/me/buckets'),
        { name },
        authHeaders()
      );
      setBuckets((prev) => [
        ...prev,
        {
          id: response.id,
          name: response.name,
          book_count: response.book_count,
          books_preview: [],
          created_at: response.created_at,
        },
      ]);
      setNewBucketName('');
      setShowCreateForm(false);
    } catch (err) {
      alert(err.message);
    } finally {
      setCreating(false);
    }
  };

  const handleRenameSubmit = async (bucketId) => {
    const name = renaming?.name?.trim();
    if (!name) return;
    setSavingRename(true);
    try {
      await usePut(
        await getBackendUrl(`/users/me/buckets/${bucketId}`),
        { name },
        authHeaders()
      );
      setBuckets((prev) =>
        prev.map((b) => (b.id === bucketId ? { ...b, name } : b))
      );
      setRenaming(null);
    } catch (err) {
      alert(err.message);
    } finally {
      setSavingRename(false);
    }
  };

  const handleDelete = async (bucketId) => {
    if (!window.confirm('Delete this bucket? This cannot be undone.')) return;
    setDeletingId(bucketId);
    try {
      await useDelete(await getBackendUrl(`/users/me/buckets/${bucketId}`), authHeaders());
      setBuckets((prev) => prev.filter((b) => b.id !== bucketId));
    } catch (err) {
      alert(err.message);
    } finally {
      setDeletingId(null);
    }
  };

  if (loading) return <div className="text-3xl font-bold text-gray-800 mb-6">Loading...</div>;
  if (error) return <div className="text-3xl font-bold text-gray-800 mb-6">Error: {error}</div>;

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-3xl font-bold text-gray-800">My Buckets</h1>
        <button
          onClick={() => { setShowCreateForm(true); setNewBucketName(''); }}
          className="px-4 py-2 bg-indigo-600 text-white text-sm font-medium rounded-md hover:bg-indigo-700 transition-colors"
        >
          + New Bucket
        </button>
      </div>

      {/* Create bucket form */}
      {showCreateForm && (
        <div className="mb-6 bg-white rounded-lg shadow-md p-5">
          <h2 className="text-sm font-semibold text-gray-700 mb-3">New Bucket</h2>
          <form onSubmit={handleCreate} className="flex items-center gap-3">
            <input
              type="text"
              value={newBucketName}
              onChange={(e) => setNewBucketName(e.target.value)}
              maxLength={40}
              placeholder="Bucket name (max 40 chars)"
              className="flex-1 border border-gray-300 rounded-md px-3 py-2 text-sm text-gray-900 bg-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-indigo-500"
              autoFocus
            />
            <button
              type="submit"
              disabled={creating || !newBucketName.trim()}
              className="px-4 py-2 bg-indigo-600 text-white text-sm font-medium rounded-md hover:bg-indigo-700 disabled:opacity-50 transition-colors"
            >
              {creating ? 'Creating...' : 'Create'}
            </button>
            <button
              type="button"
              onClick={() => setShowCreateForm(false)}
              className="px-4 py-2 bg-gray-100 text-gray-700 text-sm font-medium rounded-md hover:bg-gray-200 transition-colors"
            >
              Cancel
            </button>
          </form>
        </div>
      )}

      {/* Buckets grid */}
      {buckets.length === 0 ? (
        <div className="bg-white rounded-lg shadow-md p-12 text-center">
          <p className="text-gray-500">No buckets yet. Create one to start organising your books.</p>
        </div>
      ) : (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-5">
          {buckets.map((bucket) => (
            <div key={bucket.id} className="bg-white rounded-lg shadow-md overflow-hidden">
              {/* Book cover previews */}
              <div className="flex h-28 bg-gray-100">
                {bucket.books_preview && bucket.books_preview.length > 0 ? (
                  bucket.books_preview.map((bp) => (
                    <div key={bp.book_id} className="flex-1 overflow-hidden">
                      <img
                        src={bp.cover_image_url || `https://placehold.co/150x220/6366F1/FFFFFF?text=${encodeURIComponent(bp.title)}`}
                        alt={bp.title}
                        className="w-full h-full object-cover"
                      />
                    </div>
                  ))
                ) : (
                  <div className="flex-1 flex items-center justify-center text-gray-400">
                    <svg xmlns="http://www.w3.org/2000/svg" width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
                      <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"></path>
                    </svg>
                  </div>
                )}
              </div>

              <div className="p-4">
                {/* Bucket name / rename */}
                {renaming?.bucketId === bucket.id ? (
                  <div className="flex items-center gap-2 mb-1">
                    <input
                      type="text"
                      value={renaming.name}
                      onChange={(e) => setRenaming({ ...renaming, name: e.target.value })}
                      maxLength={40}
                      className="flex-1 border border-gray-300 rounded-md px-2 py-1 text-sm text-gray-900 bg-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-indigo-500"
                      autoFocus
                      onKeyDown={(e) => {
                        if (e.key === 'Enter') handleRenameSubmit(bucket.id);
                        if (e.key === 'Escape') setRenaming(null);
                      }}
                    />
                    <button
                      onClick={() => handleRenameSubmit(bucket.id)}
                      disabled={savingRename}
                      className="text-xs text-indigo-600 hover:text-indigo-900 font-medium disabled:opacity-50"
                    >
                      Save
                    </button>
                    <button
                      onClick={() => setRenaming(null)}
                      className="text-xs text-gray-500 hover:text-gray-700"
                    >
                      Cancel
                    </button>
                  </div>
                ) : (
                  <h3 className="font-semibold text-gray-800 truncate">{bucket.name}</h3>
                )}

                <p className="text-xs text-gray-500 mt-1">
                  {bucket.book_count ?? 0} {bucket.book_count === 1 ? 'book' : 'books'}
                </p>

                {/* Actions */}
                <div className="flex items-center gap-3 mt-3">
                  <button
                    onClick={() => setRenaming({ bucketId: bucket.id, name: bucket.name })}
                    className="text-xs text-indigo-600 hover:text-indigo-900 font-medium"
                  >
                    Rename
                  </button>
                  <button
                    onClick={() => handleDelete(bucket.id)}
                    disabled={deletingId === bucket.id}
                    className="text-xs text-red-600 hover:text-red-900 font-medium disabled:opacity-50"
                  >
                    {deletingId === bucket.id ? 'Deleting...' : 'Delete'}
                  </button>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
};

export default MyBucketsPage;
