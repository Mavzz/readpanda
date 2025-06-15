import React, { useState } from 'react';

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

// --- ICONS (as inline SVGs for portability) ---
const HomeIcon = (props) => (
  <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" {...props}>
    <path d="m3 9 9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"></path><polyline points="9 22 9 12 15 12 15 22"></polyline>
  </svg>
);
const BookIcon = (props) => (
  <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" {...props}>
    <path d="M4 19.5v-15A2.5 2.5 0 0 1 6.5 2H20v20H6.5a2.5 2.5 0 0 1 0-5H20"></path>
  </svg>
);
const UploadIcon = (props) => (
  <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" {...props}>
    <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"></path><polyline points="17 8 12 3 7 8"></polyline><line x1="12" x2="12" y1="3" y2="15"></line>
  </svg>
);
const LogoutIcon = (props) => (
    <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" {...props}>
        <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"></path><polyline points="16 17 21 12 16 7"></polyline><line x1="21" x2="9" y1="12" y2="12"></line>
    </svg>
);
const MenuIcon = (props) => (
    <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" {...props}>
        <line x1="4" x2="20" y1="12" y2="12"></line><line x1="4" x2="20" y1="6" y2="6"></line><line x1="4" x2="20" y1="18" y2="18"></line>
    </svg>
);
const SparklesIcon = (props) => (
    <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" {...props}>
        <path d="m12 3-1.9 4.8-4.8 1.9 4.8 1.9L12 21l1.9-4.8 4.8-1.9-4.8-1.9L12 3z" /><path d="M5 3v4" /><path d="M19 17v4" /><path d="M3 5h4" /><path d="M17 19h4" />
    </svg>
);


// --- Reusable Components ---
const StatCard = ({ title, value, icon, change }) => (
  <div className="bg-white p-6 rounded-lg shadow-md">
    <div className="flex items-center justify-between">
      <div>
        <p className="text-sm font-medium text-gray-500">{title}</p>
        <p className="text-2xl font-bold text-gray-800">{value}</p>
      </div>
      <div className="bg-indigo-100 text-indigo-600 p-3 rounded-full">
        {icon}
      </div>
    </div>
    {change && <p className="text-sm text-gray-500 mt-2">{change}</p>}
  </div>
);

const SuggestionModal = ({ title, suggestions, onSelect, onClose, isLoading }) => (
    <div className="fixed inset-0 bg-black bg-opacity-50 z-40 flex items-center justify-center p-4">
        <div className="bg-white rounded-lg shadow-xl max-w-lg w-full">
            <div className="p-6 border-b border-gray-200">
                <h3 className="text-lg font-medium leading-6 text-gray-900">{title}</h3>
            </div>
            <div className="p-6 max-h-96 overflow-y-auto">
                {isLoading ? (
                    <div className="flex justify-center items-center h-48">
                        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-indigo-600"></div>
                    </div>
                ) : (
                    <ul className="space-y-3">
                        {suggestions.map((suggestion, index) => (
                            <li key={index}
                                className="p-3 bg-gray-50 rounded-md cursor-pointer hover:bg-indigo-100 transition-colors"
                                onClick={() => onSelect(suggestion)}
                            >
                                <p className="text-sm text-gray-800">{suggestion}</p>
                            </li>
                        ))}
                    </ul>
                )}
            </div>
            <div className="px-4 py-3 bg-gray-50 text-right sm:px-6">
                <button
                    type="button"
                    onClick={onClose}
                    className="inline-flex justify-center py-2 px-4 border border-gray-300 shadow-sm text-sm font-medium rounded-md text-gray-700 bg-white hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500"
                >
                    Close
                </button>
            </div>
        </div>
    </div>
);


// --- Page Components ---
const DashboardPage = () => (
  <div>
    <h1 className="text-3xl font-bold text-gray-800 mb-6">Dashboard</h1>
    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
      <StatCard title="Total Books" value={mockBooks.length} icon={<BookIcon />} />
      <StatCard title="Total Views" value="29.2k" icon={<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M2 12s3-7 10-7 10 7 10 7-3 7-10 7-10-7-10-7Z"></path><circle cx="12" cy="12" r="3"></circle></svg>} change="+15% this month" />
      <StatCard title="Monthly Earnings" value="$1,451.60" icon={<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><line x1="12" x2="12" y1="2" y2="22"></line><path d="M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"></path></svg>} change="+8.2% vs last month" />
    </div>
    <div className="mt-8 bg-white p-6 rounded-lg shadow-md">
      <h2 className="text-xl font-bold text-gray-700 mb-4">Views This Year</h2>
      <div className="h-64 bg-gray-100 rounded-md flex items-center justify-center">
        <p className="text-gray-500">[Chart Data Would Be Visualized Here]</p>
      </div>
    </div>
  </div>
);

const MyBooksPage = () => (
    <div>
      <h1 className="text-3xl font-bold text-gray-800 mb-6">My Books</h1>
      <div className="bg-white rounded-lg shadow-md overflow-hidden">
        <table className="min-w-full divide-y divide-gray-200">
          <thead className="bg-gray-50">
            <tr>
              <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Book</th>
              <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Status</th>
              <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Views</th>
              <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Earnings</th>
              <th scope="col" className="relative px-6 py-3"><span className="sr-only">Actions</span></th>
            </tr>
          </thead>
          <tbody className="bg-white divide-y divide-gray-200">
            {mockBooks.map(book => (
              <tr key={book.id}>
                <td className="px-6 py-4 whitespace-nowrap">
                  <div className="flex items-center">
                    <div className="flex-shrink-0 h-20 w-14">
                      <img className="h-20 w-14 rounded object-cover" src={book.cover} alt="" />
                    </div>
                    <div className="ml-4">
                      <div className="text-sm font-medium text-gray-900">{book.title}</div>
                    </div>
                  </div>
                </td>
                <td className="px-6 py-4 whitespace-nowrap">
                  <span className={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full ${book.status === 'Live' ? 'bg-green-100 text-green-800' : 'bg-yellow-100 text-yellow-800'}`}>
                    {book.status}
                  </span>
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{book.views.toLocaleString()}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">${book.earnings.toFixed(2)}</td>
                <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                  <a href="#" className="text-indigo-600 hover:text-indigo-900 mr-4">Edit</a>
                  <a href="#" className="text-red-600 hover:text-red-900">Delete</a>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
);

const UploadBookPage = () => {
    // Form state
    const [title, setTitle] = useState('');
    const [description, setDescription] = useState('');
    const [genre, setGenre] = useState('Science Fiction');
    const [coverPreview, setCoverPreview] = useState(null);
    const [manuscriptName, setManuscriptName] = useState('');

    // Gemini AI state
    const [isGenerating, setIsGenerating] = useState({ title: false, description: false });
    const [showTitleModal, setShowTitleModal] = useState(false);
    const [titleSuggestions, setTitleSuggestions] = useState([]);

    /**
     * Calls the Gemini API to generate content.
     * @param {string} prompt The prompt to send to the model.
     * @returns {Promise<string|null>} The generated text.
     */
    const generateWithGemini = async (prompt) => {
        const apiKey = ""; // This will be provided by the runtime environment
        const apiUrl = `https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash:generateContent?key=${apiKey}`;

        try {
            const response = await fetch(apiUrl, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ contents: [{ parts: [{ text: prompt }] }] })
            });
            if (!response.ok) {
                throw new Error(`API call failed with status: ${response.status}`);
            }
            const result = await response.json();
            return result.candidates?.[0]?.content?.parts?.[0]?.text || null;
        } catch (error) {
            console.error("Gemini API call error:", error);
            alert("An error occurred while generating content. Please try again.");
            return null;
        }
    };

    const handleGenerateTitles = async () => {
        if (!description) {
            alert("Please write a description first to generate title ideas.");
            return;
        }
        setShowTitleModal(true);
        setIsGenerating(prev => ({ ...prev, title: true }));
        setTitleSuggestions([]);

        const prompt = `Based on the following book description and genre, generate a list of 5 creative and catchy book titles. Format the output as a simple list separated by newlines.\n\nGenre: ${genre}\n\nDescription: "${description}"`;
        const result = await generateWithGemini(prompt);
        
        if (result) {
            setTitleSuggestions(result.split('\n').filter(t => t.trim() !== ''));
        }
        setIsGenerating(prev => ({ ...prev, title: false }));
    };

    const handleGenerateDescription = async () => {
        if (!title) {
            alert("Please enter a title first to generate a description.");
            return;
        }
        setIsGenerating(prev => ({ ...prev, description: true }));

        const prompt = `Based on the following book title and genre, write a compelling, professional book description of about 150 words. Do not include the title in the description text itself.\n\nTitle: "${title}"\n\nGenre: ${genre}`;
        const result = await generateWithGemini(prompt);

        if (result) {
            setDescription(result);
        }
        setIsGenerating(prev => ({ ...prev, description: false }));
    };

    const handleSelectTitle = (selectedTitle) => {
        // Remove numbering like "1. " from the start of the title if it exists
        setTitle(selectedTitle.replace(/^\d+\.\s*/, ''));
        setShowTitleModal(false);
    };

    const handleCoverChange = (e) => {
        if (e.target.files && e.target.files[0]) {
            setCoverPreview(URL.createObjectURL(e.target.files[0]));
        }
    }
    const handleManuscriptChange = (e) => {
        if (e.target.files && e.target.files[0]) {
            setManuscriptName(e.target.files[0].name);
        }
    }

    return (
        <div>
            {showTitleModal && (
                <SuggestionModal
                    title="✨ AI-Generated Titles"
                    suggestions={titleSuggestions}
                    onSelect={handleSelectTitle}
                    onClose={() => setShowTitleModal(false)}
                    isLoading={isGenerating.title}
                />
            )}
            <h1 className="text-3xl font-bold text-gray-800 mb-6">Upload New Book</h1>
            <form className="space-y-8 bg-white p-8 rounded-lg shadow-md">
                <div className="grid grid-cols-1 md:grid-cols-2 gap-8">
                    {/* Left Column */}
                    <div>
                        <div>
                            <label htmlFor="title" className="block text-sm font-medium text-gray-700">Book Title</label>
                            <input type="text" name="title" id="title" value={title} onChange={(e) => setTitle(e.target.value)} className="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm" />
                        </div>

                        <div className="mt-6">
                            <div className="flex justify-between items-center">
                                <label htmlFor="description" className="block text-sm font-medium text-gray-700">Description</label>
                                <button type="button" onClick={handleGenerateDescription} disabled={isGenerating.description} className="flex items-center text-xs text-indigo-600 font-semibold hover:text-indigo-800 disabled:opacity-50 disabled:cursor-not-allowed">
                                    <SparklesIcon className="h-4 w-4 mr-1"/> {isGenerating.description ? 'Drafting...' : '✨ Draft with AI'}
                                </button>
                            </div>
                            <textarea id="description" name="description" rows="4" value={description} onChange={(e) => setDescription(e.target.value)} className="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm"></textarea>
                        </div>
                         <div className="mt-6">
                            <label htmlFor="genre" className="block text-sm font-medium text-gray-700">Genre</label>
                            <select id="genre" name="genre" value={genre} onChange={(e) => setGenre(e.target.value)} className="mt-1 block w-full pl-3 pr-10 py-2 text-base border-gray-300 focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm rounded-md">
                                <option>Science Fiction</option>
                                <option>Fantasy</option>
                                <option>Thriller</option>
                                <option>Romance</option>
                                <option>Non-Fiction</option>
                            </select>
                        </div>
                        <div className="mt-6">
                             <button type="button" onClick={handleGenerateTitles} disabled={isGenerating.title} className="w-full flex items-center justify-center py-2 px-4 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-indigo-600 hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 disabled:bg-indigo-400">
                                <SparklesIcon className="h-5 w-5 mr-2"/> ✨ Suggest Titles with AI
                             </button>
                             <p className="text-xs text-center text-gray-500 mt-2">Uses the description and genre to suggest titles.</p>
                        </div>
                    </div>

                    {/* Right Column */}
                    <div className="space-y-6">
                         <div>
                            <label className="block text-sm font-medium text-gray-700">Cover Image</label>
                            <div className="mt-1 flex items-center space-x-6">
                                <div className="flex-shrink-0 h-48 w-32 rounded-md bg-gray-100 flex items-center justify-center">
                                    {coverPreview ? 
                                        <img src={coverPreview} alt="Cover preview" className="h-full w-full object-cover rounded-md"/> :
                                        <span className="text-gray-400 text-xs text-center">Image Preview</span>
                                    }
                                </div>
                                <label htmlFor="cover-upload" className="cursor-pointer bg-white py-2 px-3 border border-gray-300 rounded-md shadow-sm text-sm leading-4 font-medium text-gray-700 hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500">
                                    <span>Upload file</span>
                                    <input id="cover-upload" name="cover-upload" type="file" className="sr-only" onChange={handleCoverChange} accept="image/*"/>
                                </label>
                            </div>
                        </div>
                        <div>
                            <label className="block text-sm font-medium text-gray-700">Book Manuscript (PDF, EPUB)</label>
                             <div className="mt-1 flex justify-center px-6 pt-5 pb-6 border-2 border-gray-300 border-dashed rounded-md">
                                <div className="space-y-1 text-center">
                                    <svg className="mx-auto h-12 w-12 text-gray-400" stroke="currentColor" fill="none" viewBox="0 0 48 48" aria-hidden="true"><path d="M28 8H12a4 4 0 00-4 4v20m32-12v8m0 0v8a4 4 0 01-4 4H12a4 4 0 01-4-4v-4m32-4l-3.172-3.172a4 4 0 00-5.656 0L28 28M8 32l9.172-9.172a4 4 0 015.656 0L28 28m0 0l4 4m4-24h8m-4-4v8" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" /></svg>
                                    <div className="flex text-sm text-gray-600">
                                        <label htmlFor="file-upload" className="relative cursor-pointer bg-white rounded-md font-medium text-indigo-600 hover:text-indigo-500 focus-within:outline-none focus-within:ring-2 focus-within:ring-offset-2 focus-within:ring-indigo-500">
                                            <span>Upload a file</span>
                                            <input id="file-upload" name="file-upload" type="file" className="sr-only" onChange={handleManuscriptChange} accept=".pdf,.epub"/>
                                        </label>
                                        <p className="pl-1">or drag and drop</p>
                                    </div>
                                    <p className="text-xs text-gray-500">{manuscriptName || 'PDF, EPUB up to 50MB'}</p>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>

                <div className="pt-5">
                    <div className="flex justify-end">
                        <button type="button" className="bg-white py-2 px-4 border border-gray-300 rounded-md shadow-sm text-sm font-medium text-gray-700 hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500">
                            Save as Draft
                        </button>
                        <button type="submit" className="ml-3 inline-flex justify-center py-2 px-4 border border-transparent shadow-sm text-sm font-medium rounded-md text-white bg-indigo-600 hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500">
                            Publish Book
                        </button>
                    </div>
                </div>
            </form>
        </div>
    )
};


// --- Main App Layout & Routing ---
const PortalLayout = ({ user, onLogout, children, setPage, currentPage }) => {
  const [sidebarOpen, setSidebarOpen] = useState(false);

  const NavLink = ({ page, icon, children: linkChildren }) => (
    <button
      onClick={() => {
        setPage(page);
        setSidebarOpen(false);
      }}
      className={`flex items-center w-full px-4 py-2 text-sm font-medium rounded-md transition-colors duration-150 ${
        currentPage === page
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
          <NavLink page="dashboard" icon={<HomeIcon />}>Dashboard</NavLink>
          <NavLink page="my-books" icon={<BookIcon />}>My Books</NavLink>
          <NavLink page="upload" icon={<UploadIcon />}>Upload Book</NavLink>
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


const LoginPage = ({ onLogin }) => (
    <div className="w-screen h-screen bg-gray-100 flex items-center justify-center py-12 px-4 sm:px-6 lg:px-8">
        <div className="max-w-md w-full space-y-8">
            <div>
                <h1 className="text-center text-4xl font-extrabold text-gray-900">
                    ReadPanda
                </h1>
                <h2 className="mt-2 text-center text-2xl font-bold text-indigo-600">
                    Writer's Portal
                </h2>
                <p className="mt-2 text-center text-sm text-gray-600">
                    Sign in to your account
                </p>
            </div>
            <form className="mt-8 space-y-6" onSubmit={(e) => { e.preventDefault(); onLogin(); }}>
                <div className="rounded-md shadow-sm -space-y-px">
                    <div>
                        <label htmlFor="email-address" className="sr-only">Email address</label>
                        <input id="email-address" name="email" type="email" autoComplete="email" required className="appearance-none rounded-none relative block w-full px-3 py-2 border border-gray-300 placeholder-gray-500 text-gray-900 rounded-t-md focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 focus:z-10 sm:text-sm" placeholder="Email address" defaultValue="writer@example.com" />
                    </div>
                    <div>
                        <label htmlFor="password" className="sr-only">Password</label>
                        <input id="password" name="password" type="password" autoComplete="current-password" required className="appearance-none rounded-none relative block w-full px-3 py-2 border border-gray-300 placeholder-gray-500 text-gray-900 rounded-b-md focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 focus:z-10 sm:text-sm" placeholder="Password" defaultValue="password123" />
                    </div>
                </div>

                <div className="flex items-center justify-between">
                    <div className="flex items-center">
                        <input id="remember-me" name="remember-me" type="checkbox" className="h-4 w-4 text-indigo-600 focus:ring-indigo-500 border-gray-300 rounded" />
                        <label htmlFor="remember-me" className="ml-2 block text-sm text-gray-900"> Remember me </label>
                    </div>

                    <div className="text-sm">
                        <a href="#" className="font-medium text-indigo-600 hover:text-indigo-500"> Forgot your password? </a>
                    </div>
                </div>

                <div>
                    <button type="submit" className="group relative w-full flex justify-center py-2 px-4 border border-transparent text-sm font-medium rounded-md text-white bg-indigo-600 hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500">
                        Sign in
                    </button>
                </div>
            </form>
        </div>
    </div>
);


// --- Main App Component ---
export default function App() {
  const [isLoggedIn, setIsLoggedIn] = useState(false);
  const [currentPage, setCurrentPage] = useState('dashboard');

  const handleLogin = () => setIsLoggedIn(true);
  const handleLogout = () => setIsLoggedIn(false);

  // Simple router
  const renderPage = () => {
    switch (currentPage) {
      case 'dashboard':
        return <DashboardPage />;
      case 'my-books':
        return <MyBooksPage />;
      case 'upload':
        return <UploadBookPage />;
      default:
        return <DashboardPage />;
    }
  };

  if (!isLoggedIn) {
    return <LoginPage onLogin={handleLogin} />;
  }

  return (
    <PortalLayout user={mockUser} onLogout={handleLogout} setPage={setCurrentPage} currentPage={currentPage}>
      {renderPage()}
    </PortalLayout>
  );
}
