import React from 'react';
import { useNavigate } from 'react-router-dom';
import DataTable from '../components/dataTable';

const UsersPage = () => {
  const navigate = useNavigate();

  // Any cell in a user row opens that user's detail page.
  const cellLink = (column, row) => (column === 'uuid' || column === 'username' || column === 'email')
    ? () => navigate(`/users/${row.uuid}`)
    : null;

  return (
    <div>
      <h1 className="text-2xl font-bold text-gray-800 mb-6">Users</h1>
      <DataTable table="users" cellLink={cellLink} />
    </div>
  );
};

export default UsersPage;
