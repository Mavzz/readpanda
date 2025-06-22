// packages/api/Service/firebaseStorageService.js
import { initializeApp } from "firebase/app";
import { getStorage } from "firebase/storage";
import admin from 'firebase-admin';
import dotenv from 'dotenv';
import path from 'path';
import { fileURLToPath } from 'url';

dotenv.config({path: './.env.local'});

let app;

const firebaseConfig = {
  apiKey: process.env.FIREBASE_API_KEY,
  authDomain: process.env.FIREBASE_AUTH_DOMAIN,
  projectId: process.env.FIREBASE_PROJECT_ID,
  storageBucket: process.env.FIREBASE_STORAGE_BUCKET,
  messagingSenderId: process.env.FIREBASE_MESSAGING_SENDER_ID,
  appId: process.env.FIREBASE_APP_ID,
  measurementId: process.env.FIREBASE_MEASUREMENT_ID
};


try {

  // Initialize Firebase
  app = initializeApp(firebaseConfig);
  console.log("Firebase Admin SDK initialized successfully.");
  
} catch (error) {
  console.error("Firebase Admin SDK initialization failed:", error);
  // It's crucial to handle this error properly in production
  // e.g., exit the process or mark the service as unavailable
}


// Initialize Cloud Storage and get a reference to the service
const storage = getStorage(app);


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