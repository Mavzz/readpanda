// packages/api/Service/firebaseStorageService.js
import admin from 'firebase-admin';
import dotenv from 'dotenv';
import path from 'path';
import { fileURLToPath } from 'url';

dotenv.config();

// Fix for __dirname in ES modules
const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

// IMPORTANT: Replace with the actual relative path to your Firebase service account key
// This file should NEVER be committed to version control directly.
// The .gitignore in packages/api already lists it as ignored:
// Firebase/Credentials/serviceAccountKey.json
const serviceAccountPath = path.resolve(__dirname, '../../api/firebase/credentials/serviceAccountKey.json');

try {
  // Initialize Firebase Admin SDK if not already initialized
  if (!admin.apps.length) {
    admin.initializeApp({
      credential: admin.credential.cert(serviceAccountPath),
      storageBucket: process.env.FIREBASE_STORAGE_BUCKET // Your Firebase Storage Bucket URL
    });
    console.log("Firebase Admin SDK initialized successfully.");
  }
} catch (error) {
  console.error("Firebase Admin SDK initialization failed:", error);
  // It's crucial to handle this error properly in production
  // e.g., exit the process or mark the service as unavailable
}

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

  const fileUpload = bucket.file(destinationPath);

  // Create a write stream to upload the file
  const stream = fileUpload.createWriteStream({
    metadata: {
      contentType: file.mimetype,
    },
  });

  return new Promise((resolve, reject) => {
    stream.on('error', (err) => {
      console.error("Error uploading to Firebase Storage:", err);
      reject(new Error("Failed to upload file to Firebase Storage."));
    });

    stream.on('finish', async () => {
      // Make the file publicly accessible. For production, consider signed URLs for private files.
      try {
        await fileUpload.makePublic();
        const publicUrl = process.env.FIREBASE_STORAGE_BUCKET ;
        resolve(publicUrl);
      } catch (publicError) {
        console.error("Error making file public:", publicError);
        reject(new Error("File uploaded but failed to set public access."));
      }
    });

    stream.end(file.buffer); // End the stream with the file buffer
  });
};