import React, { useState } from 'react';
import { BookIcon, HomeIcon, UploadIcon, LogoutIcon, MenuIcon, SparklesIcon } from './components/icons';
import LoginPage from './pages/loginPage';
import DashboardPage from './pages/dashboardPage';
import MyBooksPage from './pages/myBooksPage';
import UploadBookPage from './pages/uploadBookPage';
import { Routes, Route, Link, useNavigate } from 'react-router-dom'; // Import routing components

// --- MOCK DATA ---
// In a real application, this data would come from your API
const mockUser = {
  name: 'Jane Doe',
  email: 'jane.doe@example.com',
  avatar: 'https://placehold.co/100x100/E2E8F0/4A5568?text=JD'
};

const mockBooks = [
  { id: 1, title: 'Echoes of the Void', status: 'Live', views: 10234, earnings: 450.20, cover: 'https://placehold.co/150x220/6366F1/FFFFFF?text=Echoes' },
  { id: 2, title: 'The Last Cypher', status: 'Live', views: 8765, earnings: 320.50, cover: 'https://placehold.co/150x220/EC4899/FFFFFF?text=Cypher' },
  { id: 3, title: 'Chronicles of the Sunstone', status: 'Draft', views: 0, earnings: 0, cover: 'https://placehold.co/150x220/F59E0B/FFFFFF?text=Chronicles' },
  { id: 4, title: 'River of Whispers', status: 'Live', views: 15432, earnings: 680.90, cover: 'https://placehold.co/150x220/10B981/FFFFFF?text=River' },
];

// --- Main App Layout & Routing ---
const PortalLayout = ({ user, onLogout, children }) => {
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const navigate = useNavigate(); // Hook for programmatic navigation

  const NavLink = ({ to, icon, children: linkChildren }) => (
    <button
      onClick={() => {
        navigate(to); // Use navigate from react-router-dom
        setSidebarOpen(false);
      }}
      className={`flex items-center w-full px-4 py-2 text-sm font-medium rounded-md transition-colors duration-150 ${
        window.location.pathname === to
          ? 'bg-indigo-600 text-white'
          : 'text-gray-300 hover:bg-gray-700 hover:text-white'
      }`}
    >
      {icon}
      <span className="ml-4">{linkChildren}</span>
    </button>
  );

  return (
    <div className="flex h-screen w-screen h-screen bg-gray-100">
      {/* Sidebar */}
      <div className={`fixed inset-y-0 left-0 z-30 w-64 bg-gray-800 text-white transform ${sidebarOpen ? 'translate-x-0' : '-translate-x-full'} transition-transform duration-300 ease-in-out md:relative md:translate-x-0 md:flex md:flex-col`}>
        <div className="flex items-center justify-center h-20 border-b border-gray-700">
          <span className="text-2xl font-bold">ReadPanda</span>
        </div>
        <nav className="flex-1 px-4 py-6 space-y-2">
          {/* Use 'to' prop for React Router paths */}
          <NavLink to="/dashboard" icon={<HomeIcon />}>Dashboard</NavLink>
          <NavLink to="/my-books" icon={<BookIcon />}>My Books</NavLink>
          <NavLink to="/upload" icon={<UploadIcon />}>Upload Book</NavLink>
        </nav>
        <div className="px-4 py-4 border-t border-gray-700">
            <button
                onClick={onLogout}
                className="flex items-center w-full px-4 py-2 text-sm font-medium text-gray-300 rounded-md hover:bg-gray-700 hover:text-white"
            >
                <LogoutIcon />
                <span className="ml-4">Logout</span>
            </button>
        </div>
      </div>

      {/* Main content */}
      <div className="flex-1 flex flex-col overflow-hidden">
        {/* Top bar */}
        <header className="flex items-center justify-between h-20 px-6 bg-white border-b border-gray-200 md:justify-end">
            <button className="md:hidden" onClick={() => setSidebarOpen(!sidebarOpen)}>
                <MenuIcon />
            </button>
            <div className="flex items-center">
                <span className="mr-3 font-medium">{user.name}</span>
                <img className="h-10 w-10 rounded-full object-cover" src={user.avatar} alt="User avatar" />
            </div>
        </header>

        {/* Page Content */}
        <main className="flex-1 overflow-x-hidden overflow-y-auto bg-gray-100">
            <div className="container mx-auto px-6 py-8">
                {children}
            </div>
        </main>
      </div>
    </div>
  );
};

// --- Main App Component ---
export default function App() {
  const [isLoggedIn, setIsLoggedIn] = useState(false);
  const user = mockUser; // Your mock user
  const books = mockBooks; // Your mock books

  //const handleLogin = () => setIsLoggedIn(true);
  const handleLogout = () => setIsLoggedIn(false);

  if (!isLoggedIn) {
    return <LoginPage setIsLoggedIn={setIsLoggedIn} />;
  }

  return (
    <PortalLayout user={user} onLogout={handleLogout}>
      <Routes> {/* Define your routes here */}
        <Route path="/" element={<DashboardPage bookLength={books.length} />} />
        <Route path="/dashboard" element={<DashboardPage bookLength={books.length} />} />
        <Route path="/my-books" element={<MyBooksPage mockBooks={books} />} />
        <Route path="/upload" element={<UploadBookPage />} />
        {/* Add a catch-all route for 404 */}
        <Route path="*" element={<div>404 Not Found</div>} />
      </Routes>
    </PortalLayout>
  );
}
