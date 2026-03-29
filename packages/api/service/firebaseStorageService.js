// packages/api/Service/firebaseStorageService.js
import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';
import dotenv from 'dotenv';
import admin from 'firebase-admin';
import { generateUserUid } from '../utilities/helper.js';
dotenv.config({path: './.env.local'});

// Fix for __dirname in ES modules
const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const defaultServiceAccountPath = path.resolve(__dirname, '../serviceAccountKey.json');

const resolveServiceAccountPath = () => {
  const configuredPath = process.env.FIREBASE_SERVICE_ACCOUNT_PATH || process.env.GOOGLE_APPLICATION_CREDENTIALS;

  if (!configuredPath) {
    return defaultServiceAccountPath;
  }

  return path.isAbsolute(configuredPath)
    ? configuredPath
    : path.resolve(process.cwd(), configuredPath);
};

let bucket;
let firebaseInitError;
let firebaseInitialized = false;

const initializeFirebaseAdmin = () => {
  if (firebaseInitialized || admin.apps.length > 0) {
    firebaseInitialized = true;
    return true;
  }

  try {
    let credential;
    const serviceAccountJson = process.env.FIREBASE_SERVICE_ACCOUNT_JSON;

    if (serviceAccountJson) {
      let parsedServiceAccount;

      try {
        parsedServiceAccount = JSON.parse(serviceAccountJson);
      } catch {
        throw new Error('FIREBASE_SERVICE_ACCOUNT_JSON is not valid JSON.');
      }

      credential = admin.credential.cert(parsedServiceAccount);
    } else {
      const serviceAccountPath = resolveServiceAccountPath();

      if (!fs.existsSync(serviceAccountPath)) {
        throw new Error(
          `Service account file not found at '${serviceAccountPath}'. ` +
          'Set FIREBASE_SERVICE_ACCOUNT_PATH, GOOGLE_APPLICATION_CREDENTIALS, or FIREBASE_SERVICE_ACCOUNT_JSON.'
        );
      }

      credential = admin.credential.cert(serviceAccountPath);
    }

    admin.initializeApp({
      credential,
      storageBucket: process.env.FIREBASE_STORAGE_BUCKET
    });

    firebaseInitialized = true;
    console.log('Firebase Admin SDK initialized successfully.');
    return true;
  } catch (error) {
    firebaseInitError = error;
    console.error('Firebase Admin SDK initialization failed:', error);
    return false;
  }
};

const getBucketOrThrow = () => {
  const initialized = initializeFirebaseAdmin();

  if (!initialized) {
    throw new Error(
      `Firebase Storage is unavailable. ${firebaseInitError?.message || 'Initialization failed.'}`
    );
  }

  if (!bucket) {
    const storageBucket = process.env.FIREBASE_STORAGE_BUCKET;

    if (!storageBucket) {
      throw new Error('FIREBASE_STORAGE_BUCKET is not set.');
    }

    bucket = admin.storage().bucket(storageBucket);
  }

  return bucket;
};

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

  const bucket = getBucketOrThrow();

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

/**
 * Generates a signed download URL for a file in Firebase Storage.
 * @param {string} filePath - The path of the file in the bucket (e.g., 'books/covers/cover.jpg').
 * @returns {Promise<string>} A signed URL valid for 1 hour.
 */
export const getFileDownloadUrl = async (filePath) => {
  const bucket = getBucketOrThrow();
  const file = bucket.file(filePath);

  const [url] = await file.getSignedUrl({
    action: 'read',
    expires: Date.now() + 60 * 60 * 1000, // 1 hour
  });

  return url;
};

/**
 * Lists all files under a given prefix in Firebase Storage.
 * @param {string} prefix - The folder prefix (e.g., 'books/covers/').
 * @returns {Promise<Array<{name: string, url: string}>>} Array of file objects with name and signed URL.
 */
export const listFilesInFolder = async (prefix) => {
  const bucket = getBucketOrThrow();
  const [files] = await bucket.getFiles({ prefix });

  const results = await Promise.all(
    files
      .filter((file) => !file.name.endsWith('/')) // skip folder placeholders
      .map(async (file) => {
        const [url] = await file.getSignedUrl({
          action: 'read',
          expires: Date.now() + 60 * 60 * 1000,
        });
        return { name: file.name, url };
      })
  );

  return results;
};