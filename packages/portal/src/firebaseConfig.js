// packages/portal/src/firebaseConfig.js
import { initializeApp } from 'firebase/app';
import { getStorage } from 'firebase/storage'; // Import getStorage

// Your web app's Firebase configuration
// IMPORTANT: Replace these with your actual Firebase project configuration.
// It's recommended to store these securely in your frontend's environment variables.
// For Vite, you would typically use `import.meta.env.VITE_YOUR_VAR_NAME`.
const firebaseConfig = {
  apiKey: import.meta.env.VITE_FIREBASE_API_KEY,
  authDomain: import.meta.env.VITE_FIREBASE_AUTH_DOMAIN,
  projectId: import.meta.env.VITE_FIREBASE_PROJECT_ID,
  storageBucket: import.meta.env.VITE_FIREBASE_STORAGE_BUCKET,
  messagingSenderId: import.meta.env.VITE_FIREBASE_MESSAGING_SENDER_ID,
  appId: import.meta.env.VITE_FIREBASE_APP_ID,
};

// Initialize Firebase
const app = initializeApp(firebaseConfig);

// Get a reference to the Storage service
const storage = getStorage(app); // Use getStorage() here

export { storage }; // Export the storage instance