// packages/portal/src/services/useFileUpload.js

/**
 * Handles uploading a single file using multipart/form-data.
 * @param {string} url - The API endpoint URL for file upload.
 * @param {File} file - The File object to upload.
 * @param {string} fieldName - The name of the form field for the file (should match backend multer config, default 'file').
 * @param {object} headers - Optional additional headers.
 * @returns {Promise<object>} An object containing the status and response from the server.
 */
export const useFileUpload = async (url, formData, headers = {}) => {

    const response = await fetch(url, {
        method: "POST",
        headers: {
        // IMPORTANT: DO NOT set 'Content-Type': 'multipart/form-data' here.
        // The browser automatically sets it with the correct boundary when FormData is used as the body.
        ...headers, // Include any other custom headers like Authorization    
    },
    body: formData, // Use FormData as the body
    });

    if (!response.ok) {
        const errorText = await response.text();
        throw new Error( `Upload failed with status ${response.status}: ${errorText}`);
    }
    
    return { status: response.status, response: await response.json() };
}