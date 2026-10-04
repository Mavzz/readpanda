import { api } from '../services/api';
import { useConfirm } from './useConfirm';

export const tablePath = (table, suffix = '') => `/admin/tables/${encodeURIComponent(table)}${suffix}`;

// Asks first, then deletes. Resolves false when cancelled; throws on failure.
export const useDeleteRow = () => {
  const confirm = useConfirm();
  return async (table, primaryKey, row) => {
    const key = Object.fromEntries(primaryKey.map((c) => [c, row[c]]));
    const label = primaryKey.map((c) => `${c} = ${row[c]}`).join(', ');
    const ok = await confirm({
      title: 'Delete row',
      message: `${table}: ${label}\n\nRows in other tables that cascade from it are deleted too.`,
    });
    if (!ok) return false;
    await api.del(tablePath(table, '/rows'), { key });
    return true;
  };
};
