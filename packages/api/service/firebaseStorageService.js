// packages/api/Service/firebaseStorageService.js
import path from 'path';
import { fileURLToPath } from 'url';
import dotenv from 'dotenv';
import admin from 'firebase-admin';
import { generateUserUid } from '../utilities/helper.js';
dotenv.config({path: './.env.local'});

// Fix for __dirname in ES modules
const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const serviceAccountPath = path.resolve(__dirname, '../firebase/credentials/serviceAccountKey.json');

try {

  // Initialize Firebase
  admin.initializeApp({
    credential: admin.credential.cert(serviceAccountPath),
    storageBucket: process.env.FIREBASE_STORAGE_BUCKET
  });
  console.log("Firebase Admin SDK initialized successfully.");
  
} catch (error) {
  console.error("Firebase Admin SDK initialization failed:", error);
  // It's crucial to handle this error properly in production
  // e.g., exit the process or mark the service as unavailable
}


// Initialize Cloud Storage and get a reference to the service
const bucket = admin.storage().bucket();

/**
 * Uploads a file buffer to Firebase Storage.
 * @param {object} file - The file object from multer (e.g., req.file). Must contain 'buffer' and 'mimetype'.
 * @param {string} destinationPath - The full path where the file should be stored in the bucket (e.g., 'books/manuscripts/unique-filename.pdf').
 * @returns {Promise<string>} The public URL of the uploaded file.
 */
export const uploadFileToFirebase = async (file, destinationPath) => {
  if (!file || !file.buffer) {
    throw new Error("No file buffer provided for upload.");
  }

  if (!destinationPath) {
    throw new Error("No destination path provided for upload.");
  }

  const token = generateUserUid();

  const fileUpload = bucket.file(destinationPath);

  const stream = fileUpload.createWriteStream({
    metadata: {
      contentType: file.mimetype,
      metadata: {
        firebaseStorageDownloadTokens: token,
      },
    },
  });

  return new Promise((resolve, reject) => {
    stream.on('error', (error) => {
      console.error("Error uploading file to Firebase Storage:", error);
      reject(new Error("Failed to upload file to Firebase Storage."));
    });

    stream.on('finish', async () => {
      try {
        // Make the file publicly accessible
        await fileUpload.makePublic();

        // Get the public URL of the uploaded file
        // Note: The token is used to generate a download URL that includes the token for access
        // This is necessary for files that are not publicly accessible by default
        // The URL format is: https://storage.googleapis.com/v0/b/{bucket.name}/o/{encodedPath}?alt=media&token={token}
        // where {encodedPath} is the URL-encoded path of the file in the bucket
        // and {token} is the unique token generated for this file
        // This URL can be used to access the file directly

        const publicUrls = fileUpload.publicUrl();

        const publicUrl = `https://firebasestorage.googleapis.com/v0/b/${bucket.name}/o/${encodeURIComponent(destinationPath)}?alt=media&token=${token}`;
        resolve(publicUrl);
      } catch (publicError) {
        console.error("Error getting download URL:", publicError);
        reject(new Error("Failed to get download URL for uploaded file."));
      }
    });

    stream.end(file.buffer);
  });

};