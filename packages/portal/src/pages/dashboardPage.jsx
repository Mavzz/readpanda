import { useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { Banner, Button, Card, Cover, CoverStack, CreateSlot, EmptyState, Eyebrow, PageHeader, SearchInput, Section } from '../components/ui';
import { formatCount, useAllBooks, useOurPicks, useUserCount } from '../services/queries';

const RECENT_LIMIT = 10;
const PICKS_LIMIT = 3;

const Stat = ({ label, value }) => (
  <Card className="flex flex-col gap-2">
    <Eyebrow>{label}</Eyebrow>
    <span className="text-[30px] leading-none font-extrabold tracking-[-0.5px] text-ink-title">{value}</span>
  </Card>
);

const DashboardPage = () => {
  const navigate = useNavigate();
  const [query, setQuery] = useState('');
  const books = useAllBooks();
  const picks = useOurPicks();
  const users = useUserCount();

  const allBooks = books.data ?? [];
  const recent = [...allBooks].sort((a, b) => new Date(b.created_at) - new Date(a.created_at)).slice(0, RECENT_LIMIT);
  const buckets = picks.data ?? [];
  const totalViews = allBooks.reduce((sum, b) => sum + (b.views || 0), 0);
  const dash = (q) => (q.isLoading ? '…' : q.isError ? '–' : null);

  const failed = books.error || picks.error;

  return (
    <>
      <PageHeader title="Dashboard">
        <form onSubmit={(e) => { e.preventDefault(); navigate(`/all-books?q=${encodeURIComponent(query.trim())}`); }}>
          <SearchInput value={query} onChange={setQuery} placeholder="Search books by title or author" className="w-[300px] max-w-full bg-surface-1!" />
        </form>
        <Button onClick={() => navigate('/upload')}>Upload book</Button>
      </PageHeader>

      {failed && (
        <Banner
          message="Couldn't load everything on the dashboard."
          onRetry={() => { books.refetch(); picks.refetch(); users.refetch(); }}
        />
      )}

      <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
        <Stat label="Books" value={dash(books) ?? formatCount(allBooks.length)} />
        <Stat label="Readers" value={dash(users) ?? formatCount(users.data)} />
        <Stat label="Views" value={dash(books) ?? formatCount(totalViews)} />
      </div>

      <Section title="Recently uploaded" seeAllTo={allBooks.length > RECENT_LIMIT ? '/all-books' : null}>
        {!books.isLoading && recent.length === 0 ? (
          <EmptyState title="No books yet" body="Upload the first one and it shows up here." action={<Button onClick={() => navigate('/upload')}>Upload book</Button>} />
        ) : (
          // Rows bleed off the right edge.
          <div className="flex gap-4 overflow-x-auto -mr-6 lg:-mr-8 pr-6 lg:pr-8 pb-1">
            {recent.map((b) => (
              <Link key={b.id} to={`/all-books?q=${encodeURIComponent(b.title)}`} className="w-28 shrink-0 flex flex-col gap-2 no-underline">
                <Cover src={b.cover_image_url} title={b.title} className="w-28 h-[168px] rounded-cover" />
                <span className="text-[13px] leading-[1.3] font-extrabold text-ink-title truncate">{b.title}</span>
              </Link>
            ))}
          </div>
        )}
      </Section>

      <Section title={`Our picks · ${buckets.length}`} seeAllTo={buckets.length > PICKS_LIMIT ? '/our-picks' : null}>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
          {buckets.slice(0, PICKS_LIMIT).map((bucket) => (
            <Link
              key={bucket.id}
              to="/our-picks"
              className="rounded-card px-[18px] py-4 flex items-center gap-4 no-underline bg-[linear-gradient(135deg,#2a2418,#1a1810)]"
            >
              <CoverStack books={bucket.books_preview} />
              <div className="flex flex-col gap-1 min-w-0">
                <Eyebrow className="text-gold!">Curated</Eyebrow>
                <span className="text-sm leading-[1.3] font-extrabold text-ink-title truncate">{bucket.title}</span>
                <span className="text-xs font-semibold text-ink-meta">
                  {bucket.book_count ?? 0} {bucket.book_count === 1 ? 'book' : 'books'} · {bucket.is_active ? 'featured' : 'hidden'}
                </span>
              </div>
            </Link>
          ))}
          <CreateSlot label="New curated bucket" onClick={() => navigate('/our-picks?new=1')} />
        </div>
      </Section>
    </>
  );
};

export default DashboardPage;
