import { useEffect, useState } from 'react';
import { Routes, Route, NavLink, Navigate, useNavigate } from 'react-router-dom';
import { useQueryClient } from '@tanstack/react-query';
import { BookIcon, HomeIcon, UploadIcon, MenuIcon, SparklesIcon, FolderIcon, DatabaseIcon, UsersIcon } from './components/icons';
import { IconButton } from './components/ui';
import LoginPage from './pages/loginPage';
import DashboardPage from './pages/dashboardPage';
import MyBooksPage from './pages/myBooksPage';
import UploadBookPage from './pages/uploadBookPage';
import MyBucketsPage from './pages/myBucketsPage';
import SignUpPage from './pages/signUpPage';
import OurPicksPage from './pages/ourPicksPage';
import DatabasePage from './pages/databasePage';
import UsersPage from './pages/usersPage';
import UserDetailPage from './pages/userDetailPage';
import { endSession, hasSession } from './utils/session';
import { setUnauthorizedHandler } from './services/api';

const NAV = [
  { to: '/dashboard', label: 'Dashboard', Icon: HomeIcon },
  { to: '/all-books', label: 'All books', Icon: BookIcon },
  { to: '/our-picks', label: 'Our picks', Icon: SparklesIcon },
  { to: '/my-buckets', label: 'Buckets', Icon: FolderIcon },
  { to: '/upload', label: 'Upload book', Icon: UploadIcon },
  { to: '/users', label: 'Users', Icon: UsersIcon },
  { to: '/database', label: 'Database', Icon: DatabaseIcon },
];

const initials = (name = '') => name.trim().slice(0, 2).toUpperCase() || 'AD';

// The sidebar plays the tab bar's role (12b tab root): one step below the
// page, active item surface-2 with a light-gold icon, driven by router state.
const Sidebar = ({ onNavigate, onLogout }) => {
  const username = localStorage.getItem('username') ?? '';
  const avatar = localStorage.getItem('avatar');
  return (
    <div className="w-60 h-full bg-well flex flex-col px-3.5 py-5 gap-6 box-border">
      <div className="flex items-center gap-2.5 px-2">
        <img src="/assets/readpandaLogo.png" alt="" className="w-[30px] h-[30px] rounded-[9px] object-contain bg-ink-title p-[3px] box-border" />
        <span className="text-[17px] font-extrabold text-ink-title">ReadPanda</span>
      </div>
      <nav className="flex-1 flex flex-col gap-1 overflow-y-auto">
        {NAV.map((item) => (
          <NavLink
            key={item.to}
            to={item.to}
            onClick={onNavigate}
            className={({ isActive }) =>
              `flex items-center gap-3 h-11 px-3 rounded-field text-sm no-underline ${
                isActive ? 'bg-surface-2 text-ink-title font-extrabold' : 'text-ink-body font-bold hover:bg-surface-1'
              }`
            }
          >
            {({ isActive }) => (
              <>
                <item.Icon width={20} height={20} className={isActive ? 'text-link' : ''} />
                {item.label}
              </>
            )}
          </NavLink>
        ))}
      </nav>
      <div className="flex items-center gap-2.5 px-2">
        {avatar ? (
          <img src={avatar} alt="" className="w-[38px] h-[38px] rounded-full object-cover" />
        ) : (
          <div className="w-[38px] h-[38px] rounded-full bg-surface-2 text-ink-title flex items-center justify-center text-xs font-extrabold">{initials(username)}</div>
        )}
        <div className="flex flex-col gap-0.5 min-w-0">
          <span className="text-[13px] font-extrabold text-ink-title truncate">{username || 'admin'}</span>
          <button type="button" onClick={onLogout} className="p-0 bg-transparent text-left text-xs font-semibold text-ink-holder hover:text-ink-body">
            Sign out
          </button>
        </div>
      </div>
    </div>
  );
};

const PortalLayout = ({ onLogout, children }) => {
  const [sidebarOpen, setSidebarOpen] = useState(false);

  return (
    <div className="flex h-screen w-full bg-page overflow-hidden">
      {/* Desktop: fixed column. Below md: a drawer over a dimmed page. */}
      <div className="hidden md:block shrink-0">
        <Sidebar onLogout={onLogout} />
      </div>
      {sidebarOpen && (
        <div className="md:hidden fixed inset-0 z-40 flex">
          <Sidebar onNavigate={() => setSidebarOpen(false)} onLogout={onLogout} />
          <button type="button" aria-label="Close menu" className="flex-1 bg-black/60" onClick={() => setSidebarOpen(false)} />
        </div>
      )}

      <div className="flex-1 min-w-0 flex flex-col">
        <header className="md:hidden flex items-center gap-3 h-14 px-4 shrink-0">
          <IconButton label="Open menu" onClick={() => setSidebarOpen(true)}><MenuIcon width={18} height={18} /></IconButton>
          <span className="text-[17px] font-extrabold text-ink-title">ReadPanda</span>
        </header>
        <main className="flex-1 overflow-y-auto overflow-x-hidden">
          <div className="max-w-[1200px] mx-auto px-6 lg:px-8 py-7 flex flex-col gap-6">
            {children}
          </div>
        </main>
      </div>
    </div>
  );
};

export default function App() {
  // The token survives a refresh, so the session should too.
  const [isLoggedIn, setIsLoggedIn] = useState(hasSession);
  const [showSignUp, setShowSignUp] = useState(false);
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const handleLogout = () => {
    endSession();
    queryClient.clear();
    setIsLoggedIn(false);
    navigate('/');
  };

  // An expired or revoked token sends you back to sign in instead of
  // leaving every page stuck on an error.
  useEffect(() => {
    setUnauthorizedHandler(handleLogout);
    return () => setUnauthorizedHandler(null);
  });

  if (!isLoggedIn) {
    return showSignUp
      ? <SignUpPage setIsLoggedIn={setIsLoggedIn} onSwitchToLogin={() => setShowSignUp(false)} />
      : <LoginPage setIsLoggedIn={setIsLoggedIn} onSwitchToSignUp={() => setShowSignUp(true)} />;
  }

  return (
    <PortalLayout onLogout={handleLogout}>
      <Routes>
        <Route path="/" element={<Navigate to="/dashboard" replace />} />
        <Route path="/dashboard" element={<DashboardPage />} />
        <Route path="/all-books" element={<MyBooksPage />} />
        <Route path="/our-picks" element={<OurPicksPage />} />
        <Route path="/my-buckets" element={<MyBucketsPage />} />
        <Route path="/upload" element={<UploadBookPage />} />
        <Route path="/users" element={<UsersPage />} />
        <Route path="/users/:uuid" element={<UserDetailPage />} />
        <Route path="/database" element={<DatabasePage />} />
        <Route path="/database/:table" element={<DatabasePage />} />
        <Route path="*" element={<p className="text-ink-holder">Page not found.</p>} />
      </Routes>
    </PortalLayout>
  );
}
