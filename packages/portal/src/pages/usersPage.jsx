import { useNavigate } from 'react-router-dom';
import DataTable from '../components/dataTable';
import { PageHeader } from '../components/ui';

const UsersPage = () => {
  const navigate = useNavigate();

  // The identifying cells in a user row open that user's detail page.
  const cellLink = (column, row) => (column === 'uuid' || column === 'username' || column === 'email')
    ? () => navigate(`/users/${row.uuid}`)
    : null;

  return (
    <>
      <PageHeader title="Users" />
      <DataTable table="users" cellLink={cellLink} />
    </>
  );
};

export default UsersPage;
