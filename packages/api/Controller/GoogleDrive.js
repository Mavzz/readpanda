import google  from 'googleapis';

class GoogleDriveUploader {
  constructor(credentialsPath, tokenPath) {
    this.credentialsPath = credentialsPath;
    this.tokenPath = tokenPath;
    this.auth = null;
    this.drive = null;
  }

  async authenticate() {
    const credentials = require(this.credentialsPath);
    const { client_secret, client_id, redirect_uris } = credentials.installed;
    const oAuth2Client = new google.auth.OAuth2(
      client_id,
      client_secret,
      redirect_uris[0]
    );

    try {
      const token = require(this.tokenPath);
      oAuth2Client.setCredentials(token);
      this.auth = oAuth2Client;
      this.drive = google.drive({ version: 'v3', auth: this.auth });
      console.log('Successfully authenticated with Google Drive API.');
    } catch (err) {
      console.error('Error loading token or authenticating:', err);
      throw new Error('Failed to authenticate with Google Drive API.');
    }
  }

  async uploadFile(fileData, fileName) {
    if (!this.drive) {
      throw new Error('Google Drive not authenticated. Call authenticate() first.');
    }

    const fileMetadata = {
      name: fileName,
    };

    const media = {
      mimeType: 'application/octet-stream',
      body: fileData,
    };

    try {
      const file = await this.drive.files.create({
        resource: fileMetadata,
        media: media,
        fields: 'id, webViewLink',
      });
      console.log('File Id:', file.data.id);
      return file.data.webViewLink;
    } catch (err) {
      console.error('Error uploading file:', err);
      throw new Error('Failed to upload file to Google Drive.');
    }
  }
}
export {GoogleDriveUploader};