import { useNavigate } from 'react-router-dom';
import DataTable from '../components/dataTable';
import { PageHeader, Tag } from '../components/ui';
import { isSuspended } from '../services/moderation';

const UsersPage = () => {
  const navigate = useNavigate();

  // The identifying cells in a user row open that user's detail page.
  const cellLink = (column, row) => (column === 'uuid' || column === 'username' || column === 'email')
    ? () => navigate(`/users/${row.uuid}`)
    : null;

  // A suspended reader (report-block R9.6) stands out from the true/false
  // of everyone else.
  const renderCell = (column, row) => (column === 'isactive' && isSuspended(row)
    ? <Tag muted>Suspended</Tag>
    : undefined);

  return (
    <>
      <PageHeader title="Users" />
      <DataTable table="users" cellLink={cellLink} renderCell={renderCell} />
    </>
  );
};

export default UsersPage;
