import { useQuery } from '@tanstack/react-query';
import { api } from './api';

// Queries more than one page reads. Keys are shared so a mutation on one
// page (upload, curated edits) shows up on the others.

export const BOOKS_KEY = ['books-all'];
export const OUR_PICKS_KEY = ['our-picks'];

// GET /books/all — models.Book, keyed by `id` (not `book_id`).
export const useAllBooks = () =>
  useQuery({
    queryKey: BOOKS_KEY,
    queryFn: async () => (await api.get('/books/all')).books ?? [],
    staleTime: 60 * 1000,
  });

// GET /home/our-picks — with the portal header the API returns every
// bucket, inactive ones included, in sort order.
export const useOurPicks = () =>
  useQuery({
    queryKey: OUR_PICKS_KEY,
    queryFn: async () => (await api.get('/home/our-picks')).buckets ?? [],
  });

export const useUserCount = () =>
  useQuery({
    queryKey: ['users-count'],
    queryFn: async () => (await api.get('/users')).users?.length ?? 0,
  });

export const bookMeta = (book) =>
  [book.author_name, book.page_count ? `${book.page_count} pages` : null].filter(Boolean).join(' · ');

export const isPublished = (book) => book.status === 1;

export const formatCount = (n) =>
  n >= 1000 ? `${(n / 1000).toFixed(n >= 10000 ? 0 : 1).replace(/\.0$/, '')}k` : String(n);
