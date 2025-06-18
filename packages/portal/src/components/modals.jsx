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

export default SuggestionModal;