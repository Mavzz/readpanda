import React, { useState, useEffect } from 'react';
import { StatCard } from "../components/cards";
import { BookIcon } from "../components/icons";
import { useGet } from "../services/useGet";
import { getBackendUrl } from "../utils/Helper";

const DashboardPage = () => {
  const [totalBooks, setTotalBooks] = useState('–');
  const [totalUsers, setTotalUsers] = useState('–');
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchStats = async () => {
      const headers = { Authorization: `Bearer ${localStorage.getItem('token')}` };
      try {
        const [booksRes, usersRes] = await Promise.all([
          useGet(await getBackendUrl('/books/all'), headers),
          useGet(await getBackendUrl('/users'), headers),
        ]);
        setTotalBooks(booksRes.response.books?.length ?? '–');
        setTotalUsers(usersRes.response.users?.length ?? '–');
      } catch {
        // stats remain as '–'
      } finally {
        setLoading(false);
      }
    };
    fetchStats();
  }, []);

  return (
    <div>
      <h1 className="text-3xl font-bold text-gray-800 mb-6">Admin Dashboard</h1>
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        <StatCard
          title="Total Books"
          value={loading ? '…' : totalBooks}
          icon={<BookIcon />}
        />
        <StatCard
          title="Total Users"
          value={loading ? '…' : totalUsers}
          icon={<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"></path><circle cx="9" cy="7" r="4"></circle><path d="M23 21v-2a4 4 0 0 0-3-3.87"></path><path d="M16 3.13a4 4 0 0 1 0 7.75"></path></svg>}
        />
        <StatCard
          title="Total Views"
          value="29.2k"
          icon={<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M2 12s3-7 10-7 10 7 10 7-3 7-10 7-10-7-10-7Z"></path><circle cx="12" cy="12" r="3"></circle></svg>}
          change="+15% this month"
        />
      </div>
      <div className="mt-8 bg-white p-6 rounded-lg shadow-md">
        <h2 className="text-xl font-bold text-gray-700 mb-4">Views This Year</h2>
        <div className="h-64 bg-gray-100 rounded-md flex items-center justify-center">
          <p className="text-gray-500">[Chart Data Would Be Visualized Here]</p>
        </div>
      </div>
    </div>
  );
};

export default DashboardPage;