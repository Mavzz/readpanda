import React, { useState } from 'react';
import { BookIcon, HomeIcon, UploadIcon, LogoutIcon, MenuIcon, SparklesIcon, FolderIcon } from './components/icons';
import LoginPage from './pages/loginPage';
import DashboardPage from './pages/dashboardPage';
import MyBooksPage from './pages/myBooksPage';
import UploadBookPage from './pages/uploadBookPage';
import MyBucketsPage from './pages/myBucketsPage';
import SignUpPage from './pages/signUpPage';
import OurPicksPage from './pages/ourPicksPage';
import { Routes, Route, Link, useNavigate } from 'react-router-dom'; // Import routing components

const mockAvatar = 'https://placehold.co/100x100/E2E8F0/4A5568?text=AD';

// --- Main App Layout & Routing ---
const PortalLayout = ({ onLogout, children }) => {
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
          <span className="ml-2 text-xs font-semibold uppercase tracking-widest text-indigo-400 self-end mb-1">Admin</span>
        </div>
        <nav className="flex-1 px-4 py-6 space-y-2">
          {/* Use 'to' prop for React Router paths */}
          <NavLink to="/dashboard" icon={<HomeIcon />}>Dashboard</NavLink>
          <NavLink to="/all-books" icon={<BookIcon />}>All Books</NavLink>
          <NavLink to="/our-picks" icon={<SparklesIcon />}>Our Picks</NavLink>
          <NavLink to="/my-buckets" icon={<FolderIcon />}>Buckets</NavLink>
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
                <span className="mr-3 font-medium text-gray-500">{localStorage.getItem("username")}</span>
                <img className="h-10 w-10 rounded-full object-cover" src={localStorage.getItem("avatar") ? localStorage.getItem("avatar") : mockAvatar} alt="User avatar" />
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
  const [showSignUp, setShowSignUp] = useState(false);

  const handleLogout = () => setIsLoggedIn(false);

  if (!isLoggedIn) {
    return showSignUp
      ? <SignUpPage setIsLoggedIn={setIsLoggedIn} onSwitchToLogin={() => setShowSignUp(false)} />
      : <LoginPage setIsLoggedIn={setIsLoggedIn} onSwitchToSignUp={() => setShowSignUp(true)} />;
  }

  return (
    <PortalLayout onLogout={handleLogout}>
      <Routes> {/* Define your routes here */}
        <Route path="/" element={<DashboardPage />} />
        <Route path="/dashboard" element={<DashboardPage />} />
        <Route path="/all-books" element={<MyBooksPage />} />
        <Route path="/our-picks" element={<OurPicksPage />} />
        <Route path="/my-buckets" element={<MyBucketsPage />} />
        <Route path="/upload" element={<UploadBookPage />} />
        {/* Add a catch-all route for 404 */}
        <Route path="*" element={<div>404 Not Found</div>} />
      </Routes>
    </PortalLayout>
  );
}
