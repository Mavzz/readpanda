import { GoogleDriveUploader } from '../Controller/GoogleDrive.js';

export const fileuploader = async (req, res) => {
    try {
        if (!req.file) {
          return res.status(400).send({ message: "No file uploaded." });
        }
        const credentialsPath = 'credentials.json';
        const tokenPath = 'token.json';
        const uploader = new GoogleDriveUploader(credentialsPath, tokenPath);
        await uploader.authenticate();
        const fileData = req.file.buffer;
        const fileName = req.file.originalname;
    
        const fileLink = await uploader.uploadFile(fileData, fileName);
        res.status(200).send({
          message: "File uploaded successfully.",
          link: fileLink,
        });
      } catch (error) {
        console.error("Error uploading file:", error);
        res.status(500).send({
          message: "Failed to upload file.",
          error: error.message,
        });
      }
}