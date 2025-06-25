// packages/api/Controller/fileController.js
// Remove this import as it's no longer used for Firebase Storage:
// import { GoogleDriveUploader } from '../Controller/GoogleDrive.js';

// Import the new Firebase Storage service:
import { uploadFileToFirebase } from '../service/firebaseStorageService.js';
import { checkToken, decodeToken } from "../utilities/helper.js";
import client from '../database/config.js';

/**
 * Publishes the book and cover image.
 * @param {Object} req - The request object (with req.userId from middleware).
 * @param {Object} res - The response object.
 */

export const publishBook = async (req, res) => {
  const authHeader = req.header("authorization");
  const token = authHeader && authHeader.split(" ")[1]; // Bearer <token>

  try {
    if (checkToken(token, process.env.JWT_SECRET)) {
      if (!req.files || (!req.files.cover && !req.files.manuscript)) {
        return res.status(400).send({ message: "No files uploaded." });
      }

      const uuid = decodeToken(token, process.env.JWT_SECRET);
      const { title, description, genre, subgenre }  = req.body;

      let coverLink = null;
      let manuscriptLink = null;

      // Upload cover if present
      if (req.files.cover && req.files.cover[0]) {
        const coverFile = req.files.cover[0];
        const coverPath = `books/covers/${coverFile.originalname}`;
        coverLink = await uploadFileToFirebase(coverFile, coverPath);
      }

      // Upload manuscript if present
      if (req.files.manuscript && req.files.manuscript[0]) {
        const manuscriptFile = req.files.manuscript[0];
        const manuscriptPath = `books/manuscripts/${manuscriptFile.originalname}`;
        manuscriptLink = await uploadFileToFirebase(manuscriptFile, manuscriptPath);
      }

      const result = await client.query(
      "INSERT INTO books (user_id, title, description, subgenre, genre, cover_image_url, manuscript_url, status, views) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING *",
      [uuid.userId, title, description, subgenre, genre, coverLink, manuscriptLink, 0, 0]
    );

      res.status(200).json({
        message: "Book uploaded successfully"
      });
    } else {
      res.status(401).json({ error: "Unauthorized" });
    }
  } catch (error) {
      console.error("Error uploading file to Firebase Storage:", error);
      res.status(500).send({
        message: "Failed to upload file to Firebase Storage.",
        error: error.message,
      });
    }
}

/**
 * Get all books for the authenticated user.
 * @param {Object} req - The request object (with req.userId from middleware).
 * @param {Object} res - The response object.
 */

export const getBooksForUser = async (req, res) => {
  const authHeader = req.header("authorization");
  const token = authHeader && authHeader.split(" ")[1]; // Bearer <token>

  try {
    if (checkToken(token, process.env.JWT_SECRET)) {

      const uuid = decodeToken(token, process.env.JWT_SECRET);
      if (!uuid) {
        return res.status(400).json({ error: "Invalid token" });
      }
      const result = await client.query(
        "SELECT * FROM books WHERE user_id = $1",
        [uuid.userId]
      );

      res.status(200).json({books: result.rows});
    } else {
      res.status(401).json({ error: "Unauthorized" });
    }
  } catch (error) {
    console.error("Error fetching books for user:", error);
    res.status(500).json({ error: "Internal server error" });
  }
}