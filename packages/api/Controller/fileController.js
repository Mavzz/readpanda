// packages/api/Controller/fileController.js
// Remove this import as it's no longer used for Firebase Storage:
// import { GoogleDriveUploader } from '../Controller/GoogleDrive.js';

// Import the new Firebase Storage service:
import { uploadFileToFirebase } from '../service/firebaseStorageService.js';
import { checkToken } from "../utilities/helper.js";

export const fileuploader = async (req, res) => {
  const authHeader = req.header("authorization");
  const token = authHeader && authHeader.split(" ")[1]; // Bearer <token>

  try {
    if (checkToken(token, process.env.JWT_SECRET)) {
      if (!req.file) {
        return res.status(400).send({ message: "No file uploaded." });
      }
      
      const fileName = req.file.originalname;
      // Define a unique destination path in Firebase Storage
      // Using Date.now() ensures a unique filename, preventing conflicts.
      const destinationPath = `books/manuscripts/${Date.now()}_${fileName}`; // Example path

      // Call the new Firebase Storage uploader function
      const fileLink = await uploadFileToFirebase(req.file, destinationPath);

      res.status(200).send({
        message: "File uploaded successfully to Firebase Storage.",
        link: fileLink,
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