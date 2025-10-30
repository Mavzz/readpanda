import express from 'express';
import multer from 'multer';
import dotenv from 'dotenv';
import { createUser, loginUser, getUsers, googleAuth, refreshAccessToken, logoutUser } from "../controller/users.js";
import { getUserPreferences, updateUserPreferences, getGenres, getSubgenres } from "../controller/user_preferences.js";
import { publishBook, getBooksForUser, getAllBooks } from '../controller/fileController.js';
import { getUserNotifications, getUnreadNotificationCount } from "../Controller/notification.js"


dotenv.config({path: './.env.local'});

const upload = multer();
const API_VERSION = process.env.API_VERSION || '/api/v1';
const router = express.Router();

// User routes
router.get(`${API_VERSION}/users`, getUsers); // Route to get all users
router.post(`${API_VERSION}/signup`, createUser); // Route to create a new user

router.get(`${API_VERSION}/token/refresh`, refreshAccessToken);

// Authenticate user
router.post(`${API_VERSION}/auth/login`, loginUser); // Route to login a user
router.post(`${API_VERSION}/auth/google`, googleAuth); // Google authentication
router.post(`${API_VERSION}/auth/logout`, logoutUser); // Logout user

// User preferences
router.get(`${API_VERSION}/user/preferences`, getUserPreferences); // Get user preferences
router.post(`${API_VERSION}/user/preferences`, updateUserPreferences); // Update user preferences

// Books
router.post(`${API_VERSION}/books/upload`, upload.fields([{ name: 'cover', maxCount: 1 }, { name: 'manuscript', maxCount: 1 }]), publishBook); // Route to upload a file
router.get(`${API_VERSION}/books`, getBooksForUser); // Route to get books for a user
router.get(`${API_VERSION}/books/all`, getAllBooks); // Route to get all books

// Genres / Subgenres
router.get(`${API_VERSION}/genres`, getGenres); // Route to get genres
router.get(`${API_VERSION}/subgenres`, getSubgenres); // Route to get subgenres

// Notifications
router.get(`${API_VERSION}/notifications`, getUserNotifications); // Route to get user notifications
router.get(`${API_VERSION}/notifications/unread/count`, getUnreadNotificationCount); // Route to get user notifications count

export default router;