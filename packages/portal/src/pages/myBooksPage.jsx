import { useState } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { Banner, BookRow, Button, Chip, EmptyState, List, ListItem, Loading, PageHeader, SearchInput, Tag } from '../components/ui';
import { bookMeta, formatCount, isPublished, useAllBooks } from '../services/queries';

const FILTERS = [
  { id: 'all', label: 'All', test: () => true },
  { id: 'published', label: 'Published', test: isPublished },
  { id: 'draft', label: 'Draft', test: (b) => !isPublished(b) },
];

const MyBooksPage = () => {
  const navigate = useNavigate();
  const [params, setParams] = useSearchParams();
  const [filter, setFilter] = useState('all');
  const query = params.get('q') ?? '';
  const { data: books = [], isLoading, error, refetch } = useAllBooks();

  const setQuery = (q) => setParams(q ? { q } : {}, { replace: true });

  const needle = query.trim().toLowerCase();
  const test = FILTERS.find((f) => f.id === filter).test;
  const shown = books
    .filter(test)
    .filter((b) => !needle || b.title.toLowerCase().includes(needle) || (b.author_name ?? '').toLowerCase().includes(needle))
    .sort((a, b) => new Date(b.created_at) - new Date(a.created_at));

  return (
    <>
      <PageHeader title="All books" subtitle={isLoading ? null : `${books.length} ${books.length === 1 ? 'book' : 'books'}`}>
        <Button onClick={() => navigate('/upload')}>Upload book</Button>
      </PageHeader>

      <div className="flex flex-wrap items-center gap-3">
        <SearchInput value={query} onChange={setQuery} placeholder="Search books by title or author" className="flex-1 min-w-[220px] max-w-md" />
        <div className="flex gap-2">
          {FILTERS.map((f) => (
            <Chip key={f.id} selected={filter === f.id} onClick={() => setFilter(f.id)}>{f.label}</Chip>
          ))}
        </div>
      </div>

      {error ? (
        <Banner message="Couldn't load books." onRetry={refetch} />
      ) : isLoading ? (
        <Loading />
      ) : books.length === 0 ? (
        <EmptyState title="No books yet" body="Upload a manuscript and it appears here." action={<Button onClick={() => navigate('/upload')}>Upload book</Button>} />
      ) : shown.length === 0 ? (
        <p className="m-0 text-[13px] text-ink-holder">Nothing matches that search.</p>
      ) : (
        <List>
          {shown.map((book) => (
            <ListItem key={book.id}>
              <BookRow
                cover={book.cover_image_url}
                title={book.title}
                meta={bookMeta(book) || book.genre}
                tags={
                  <>
                    {isPublished(book) ? <Tag dot>Published</Tag> : <Tag muted>Draft</Tag>}
                    {book.genre && <Tag>{book.genre}</Tag>}
                  </>
                }
                trailing={<span className="text-xs font-semibold text-ink-holder whitespace-nowrap">{formatCount(book.views ?? 0)} views</span>}
              />
            </ListItem>
          ))}
        </List>
      )}
    </>
  );
};

export default MyBooksPage;
