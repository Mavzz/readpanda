import { useQuery } from '@tanstack/react-query';
import { useNavigate, useParams } from 'react-router-dom';
import { api } from '../services/api';
import DataTable from '../components/dataTable';
import { Banner, PageHeader } from '../components/ui';

// Columns that hold a user's uuid — clicking one opens that user's detail page.
const USER_COLUMNS = new Set(['user_id', 'admin_id']);

const DatabasePage = () => {
  const { table } = useParams();
  const navigate = useNavigate();

  const { data: tables = [], error, refetch } = useQuery({
    queryKey: ['admin-tables'],
    queryFn: async () => (await api.get('/admin/tables')).tables,
  });

  const cellLink = (column, row) => {
    const isUser = (table === 'users' && column === 'uuid') || USER_COLUMNS.has(column);
    return isUser && row[column] ? () => navigate(`/users/${row[column]}`) : null;
  };

  return (
    <>
      <PageHeader title="Database" />
      {error && <Banner message={error.message} onRetry={refetch} />}
      <div className="flex flex-col lg:flex-row gap-6 items-start">
        <aside className="w-full lg:w-56 shrink-0 bg-surface-1 rounded-card p-1.5 flex flex-col gap-0.5">
          {tables.map((t) => (
            <button
              key={t.name}
              type="button"
              onClick={() => navigate(`/database/${t.name}`)}
              className={`w-full flex justify-between items-center h-10 px-3 rounded-field text-left text-[13px] ${
                t.name === table ? 'bg-surface-2 text-ink-title font-extrabold' : 'bg-transparent text-ink-body font-semibold hover:bg-surface-2'
              }`}
            >
              <span className="truncate">{t.name}</span>
              <span className="ml-2 text-xs text-ink-holder">{t.row_count}</span>
            </button>
          ))}
        </aside>
        <section className="flex-1 min-w-0 w-full">
          {table ? (
            <DataTable table={table} cellLink={cellLink} />
          ) : (
            <p className="m-0 text-[13px] text-ink-holder">Pick a table to browse its rows.</p>
          )}
        </section>
      </div>
    </>
  );
};

export default DatabasePage;
