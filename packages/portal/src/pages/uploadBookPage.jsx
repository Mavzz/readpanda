import { useState } from "react";
import SuggestionModal from "../components/modals";
import { SparklesIcon } from "../components/icons";
import { useFileUpload as UseFileUpload } from "../services/useFileUpload";
import { getBackendUrl } from "../utils/Helper";

const UploadBookPage = () => {
    // Form state
    const [title, setTitle] = useState("");
    const [description, setDescription] = useState("");
    const [genre, setGenre] = useState("Science Fiction");
    const [coverPreview, setCoverPreview] = useState(null);
    const [manuscriptName, setManuscriptName] = useState("");

    // New state for file objects
    const [coverFile, setCoverFile] = useState(null);
    const [manuscriptFile, setManuscriptFile] = useState(null);

    // Gemini AI state
    const [isGenerating, setIsGenerating] = useState({
        title: false,
        description: false,
    });
    const [showTitleModal, setShowTitleModal] = useState(false);
    const [titleSuggestions, setTitleSuggestions] = useState([]);

    // Upload specific state
    const [isUploading, setIsUploading] = useState(false);
    const [uploadMessage, setUploadMessage] = useState("");
    const [uploadError, setUploadError] = useState("");

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
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ contents: [{ parts: [{ text: prompt }] }] }),
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
        setIsGenerating((prev) => ({ ...prev, title: true }));
        setTitleSuggestions([]);

        const prompt = `Based on the following book description and genre, generate a list of 5 creative and catchy book titles. Format the output as a simple list separated by newlines.\n\nGenre: ${genre}\n\nDescription: "${description}"`;
        const result = await generateWithGemini(prompt);

        if (result) {
            setTitleSuggestions(result.split("\n").filter((t) => t.trim() !== ""));
        }
        setIsGenerating((prev) => ({ ...prev, title: false }));
    };

    const handleGenerateDescription = async () => {
        if (!title) {
            alert("Please enter a title first to generate a description.");
            return;
        }
        setIsGenerating((prev) => ({ ...prev, description: true }));

        const prompt = `Based on the following book title and genre, write a compelling, professional book description of about 150 words. Do not include the title in the description text itself.\n\nTitle: "${title}"\n\nGenre: ${genre}`;
        const result = await generateWithGemini(prompt);

        if (result) {
            setDescription(result);
        }
        setIsGenerating((prev) => ({ ...prev, description: false }));
    };

    const handleSelectTitle = (selectedTitle) => {
        // Remove numbering like "1. " from the start of the title if it exists
        setTitle(selectedTitle.replace(/^\d+\.\s*/, ""));
        setShowTitleModal(false);
    };

    const handleCoverChange = (e) => {
        if (e.target.files && e.target.files[0]) {
            setCoverFile(e.target.files[0]); // Store the File object
            setCoverPreview(URL.createObjectURL(e.target.files[0]));
        } else {
            setCoverFile(null);
            setCoverPreview(null);
        }
    };
    const handleManuscriptChange = (e) => {
        if (e.target.files && e.target.files[0]) {
            setManuscriptFile(e.target.files[0]); // Store the File object
            setManuscriptName(e.target.files[0].name);
        } else {
            setManuscriptFile(null);
            setManuscriptName("");
        }
    };

    const handlePublish = async (e) => {
        e.preventDefault();

        setUploadMessage("");
        setUploadError("");

        if (!title || !description || !genre || !manuscriptFile) {
            setUploadError(
                "Please fill in all required fields and upload a manuscript file."
            );
            return;
        }

        setIsUploading(true);

        try {
            // Upload Cover Image

            const token = localStorage.getItem("token");
            const headers = {
                Authorization: `Bearer ${token}`
            };

            const coverUploadUrl = await getBackendUrl("/upload");
            const coverResponse = await UseFileUpload(
                coverUploadUrl,
                coverFile,
                "file",
                headers
            );
            console.log("Cover image upload response:", coverResponse.response.link);
            setUploadMessage(`Cover uploaded to: ${coverResponse.response.link}`);

            // Upload Book Manuscript
            const manuscriptUploadUrl = await getBackendUrl("/upload");
            const manuscriptResponse = await UseFileUpload(
                manuscriptUploadUrl,
                manuscriptFile,
                "file",
                headers
            );
            console.log(
                "Manuscript upload response:",
                manuscriptResponse.response.link
            );
            setUploadMessage(
                (prev) =>
                    prev + `Manuscript uploaded to: ${manuscriptResponse.response.link}`
            );

            setUploadMessage("Book published successfully!");
            // Clear form or redirect after successful upload
            setTitle("");
            setDescription("");
            setGenre("Science Fiction");
            setCoverFile(null);
            setCoverPreview(null);
            setManuscriptFile(null);
            setManuscriptName("");
        } catch (error) {
            console.error("Error during book upload:", error);
            setUploadError(
                "An error occurred while uploading your book. Please try again."
            );
        } finally {
            setIsUploading(false);
        }
    };

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
            <form
                className="space-y-8 bg-white p-8 rounded-lg shadow-md"
                onSubmit={handlePublish}
            >
                {" "}
                {/* Link onSubmit to handlePublish */}
                {uploadError && (
                    <div
                        className="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded relative"
                        role="alert"
                    >
                        <strong className="font-bold">Upload Error: </strong>
                        <span className="block sm:inline">{uploadError}</span>
                    </div>
                )}
                {uploadMessage && (
                    <div
                        className="bg-green-100 border border-green-400 text-green-700 px-4 py-3 rounded relative"
                        role="alert"
                    >
                        <strong className="font-bold">Success: </strong>
                        <span className="block sm:inline">{uploadMessage}</span>
                    </div>
                )}
                <div className="grid grid-cols-1 md:grid-cols-2 gap-8">
                    {/* Left Column */}
                    <div>
                        <div>
                            <label
                                htmlFor="title"
                                className="block text-sm font-medium text-gray-700"
                            >
                                Book Title
                            </label>
                            <input
                                type="text"
                                name="title"
                                id="title"
                                value={title}
                                onChange={(e) => setTitle(e.target.value)}
                                className="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm text-gray-900"
                            />
                        </div>

                        <div className="mt-6">
                            <div className="flex justify-between items-center">
                                <label
                                    htmlFor="description"
                                    className="block text-sm font-medium text-gray-700"
                                >
                                    Description
                                </label>
                                <button
                                    type="button"
                                    onClick={handleGenerateDescription}
                                    disabled={isGenerating.description}
                                    className="flex items-center text-xs text-indigo-600 font-semibold hover:text-indigo-800 disabled:opacity-50 disabled:cursor-not-allowed"
                                >
                                    <SparklesIcon className="h-4 w-4 mr-1" />{" "}
                                    {isGenerating.description
                                        ? "Drafting..."
                                        : "✨ Draft with AI"}
                                </button>
                            </div>
                            <textarea
                                id="description"
                                name="description"
                                rows="4"
                                value={description}
                                onChange={(e) => setDescription(e.target.value)}
                                className="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm text-gray-900"
                            ></textarea>
                        </div>
                        <div className="mt-6">
                            <label
                                htmlFor="genre"
                                className="block text-sm font-medium text-gray-700"
                            >
                                Genre
                            </label>
                            <select
                                id="genre"
                                name="genre"
                                value={genre}
                                onChange={(e) => setGenre(e.target.value)}
                                className="mt-1 block w-full pl-3 pr-10 py-2 text-base border-gray-300 focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm rounded-md"
                            >
                                <option>Science Fiction</option>
                                <option>Fantasy</option>
                                <option>Thriller</option>
                                <option>Romance</option>
                                <option>Non-Fiction</option>
                            </select>
                        </div>
                        <div className="mt-6">
                            <button
                                type="button"
                                onClick={handleGenerateTitles}
                                disabled={isGenerating.title}
                                className="w-full flex items-center justify-center py-2 px-4 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-indigo-600 hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 disabled:bg-indigo-400"
                            >
                                <SparklesIcon className="h-5 w-5 mr-2" /> ✨ Suggest Titles with
                                AI
                            </button>
                            <p className="text-xs text-center text-gray-500 mt-2">
                                Uses the description and genre to suggest titles.
                            </p>
                        </div>
                    </div>

                    {/* Right Column */}
                    <div className="space-y-6">
                        <div>
                            <label className="block text-sm font-medium text-gray-700">
                                Cover Image
                            </label>
                            <div className="mt-1 flex items-center space-x-6">
                                <div className="flex-shrink-0 h-48 w-32 rounded-md bg-gray-100 flex items-center justify-center">
                                    {coverPreview ? (
                                        <img
                                            src={coverPreview}
                                            alt="Cover preview"
                                            className="h-full w-full object-cover rounded-md"
                                        />
                                    ) : (
                                        <span className="text-gray-400 text-xs text-center">
                                            Image Preview
                                        </span>
                                    )}
                                </div>
                                <label
                                    htmlFor="cover-upload"
                                    className="cursor-pointer bg-white py-2 px-3 border border-gray-300 rounded-md shadow-sm text-sm leading-4 font-medium text-gray-700 hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500"
                                >
                                    <span>Upload file</span>
                                    <input
                                        id="cover-upload"
                                        name="cover-upload"
                                        type="file"
                                        className="sr-only"
                                        onChange={handleCoverChange}
                                        accept="image/*"
                                    />
                                </label>
                            </div>
                        </div>
                        <div>
                            <label className="block text-sm font-medium text-gray-700">
                                Book Manuscript (PDF, EPUB)
                            </label>
                            <div className="mt-1 flex justify-center px-6 pt-5 pb-6 border-2 border-gray-300 border-dashed rounded-md">
                                <div className="space-y-1 text-center">
                                    <svg
                                        className="mx-auto h-12 w-12 text-gray-400"
                                        stroke="currentColor"
                                        fill="none"
                                        viewBox="0 0 48 48"
                                        aria-hidden="true"
                                    >
                                        <path
                                            d="M28 8H12a4 4 0 00-4 4v20m32-12v8m0 0v8a4 4 0 01-4 4H12a4 4 0 01-4-4v-4m32-4l-3.172-3.172a4 4 0 00-5.656 0L28 28M8 32l9.172-9.172a4 4 0 015.656 0L28 28m0 0l4 4m4-24h8m-4-4v8"
                                            strokeWidth="2"
                                            strokeLinecap="round"
                                            strokeLinejoin="round"
                                        />
                                    </svg>
                                    <div className="flex text-sm text-gray-600">
                                        <label
                                            htmlFor="file-upload"
                                            className="relative cursor-pointer bg-white rounded-md font-medium text-indigo-600 hover:text-indigo-500 focus-within:outline-none focus-within:ring-2 focus-within:ring-offset-2 focus-within:ring-indigo-500"
                                        >
                                            <span>Upload a file</span>
                                            <input
                                                id="file-upload"
                                                name="file-upload"
                                                type="file"
                                                className="sr-only"
                                                onChange={handleManuscriptChange}
                                                accept=".pdf,.epub"
                                            />
                                        </label>
                                        <p className="pl-1">or drag and drop</p>
                                    </div>
                                    <p className="text-xs text-gray-500">
                                        {manuscriptName || "PDF, EPUB up to 50MB"}
                                    </p>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
                <div className="pt-5">
                    <div className="flex justify-end">
                        <button
                            type="button"
                            className="bg-white py-2 px-4 border border-gray-300 rounded-md shadow-sm text-sm font-medium text-gray-700 hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500"
                        >
                            Save as Draft
                        </button>
                        <button
                            type="submit" // This button will now trigger handlePublish via form onSubmit
                            className="ml-3 inline-flex justify-center py-2 px-4 border border-transparent shadow-sm text-sm font-medium rounded-md text-white bg-indigo-600 hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500"
                            disabled={
                                isUploading || isGenerating.title || isGenerating.description
                            } // Disable during any processing
                        >
                            {isUploading ? "Publishing..." : "Publish Book"}
                        </button>
                    </div>
                </div>
            </form>
        </div>
    );
};

export default UploadBookPage;
