import express from 'express';
import { createUser, loginUser, getUsers, googleAuth } from "../Controller/users.js";
import { getUserPreferences, updateUserPreferences } from "../Controller/user_preferences.js";
import { fileuploader } from '../Controller/fileController.js';
import multer from 'multer';
const upload = multer();

const router = express.Router();

// Route to get all users
router.get('/users', getUsers);

// Route to create a new user
router.post("/signup", createUser);

// Route to login a user
router.post("/auth/login", loginUser);

// Get user preferences
router.get("/user/preferences", getUserPreferences);

router.post("/user/preferences", updateUserPreferences);

router.post("/auth/google", googleAuth); // Placeholder for Google authentication, implement as needed

// Route to upload a file
router.post("/upload", upload.single('file'), fileuploader);
export default router;