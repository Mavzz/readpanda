// packages/api/Service/firebaseStorageService.js
import { initializeApp } from "firebase/app";
import { getStorage, ref, uploadBytes, getDownloadURL } from "firebase/storage";
import dotenv from 'dotenv';
import admin from 'firebase-admin';

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

  if (!destinationPath) {
    throw new Error("No destination path provided for upload.");
  }

  const storageRef = ref(storage, destinationPath);
  const snapshot = await uploadBytes(storageRef, file);
  const publicUrl = await getDownloadURL(snapshot.ref);

  return publicUrl;
};

export const uploadFileToFirebases = async (file, destinationPath) => {
  if (!file || !file.buffer) {
    throw new Error("No file buffer provided for upload.");
  }

  if (!destinationPath) {
    throw new Error("No destination path provided for upload.");
  }

  const fileUpload = bucket.file(destinationPath);

  const stream = fileUpload.createWriteStream({
    metadata: {
      contentType: file.mimetype,
    },
  });

  return new Promise((resolve, reject) => {
    stream.on('error', (error) => {
      console.error("Error uploading file to Firebase Storage:", error);
      reject(new Error("Failed to upload file to Firebase Storage."));
    });

    stream.on('finish', async () => {
      try {
        const publicUrl = `https://storage.googleapis.com/${bucket.name}/${fileUpload.name}`;
        resolve(publicUrl);
      } catch (publicError) {
        console.error("Error getting download URL:", publicError);
        reject(new Error("Failed to get download URL for uploaded file."));
      }
    });

    stream.end(file.buffer);
  });

};