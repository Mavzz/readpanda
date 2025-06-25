import express from 'express';
import { createUser, loginUser, getUsers, googleAuth } from "../controller/users.js";
import { getUserPreferences, updateUserPreferences, getGenres, getSubgenres } from "../controller/user_preferences.js";
import { publishBook, getBooksForUser, getAllBooks } from '../controller/fileController.js';
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
router.post("/books/upload", upload.fields([{ name: 'cover', maxCount: 1 }, { name: 'manuscript', maxCount: 1 }]), publishBook);

// New route to get genres
router.get("/genres", getGenres);

// New route to get subgenres
router.get("/subgenres", getSubgenres);

// Route to get books for a user
router.get("/books", getBooksForUser);

router.get("/allbooks", getAllBooks);

export default router;