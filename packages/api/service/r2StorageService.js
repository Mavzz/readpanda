// packages/api/service/r2StorageService.js
//
// S3-compatible storage service for Cloudflare R2 and local MinIO.
// Switch between them purely via environment variables — no code changes needed.
//
// Local dev  → R2_ENDPOINT=http://localhost:9000  (MinIO via docker-compose)
// Production → R2_ENDPOINT=https://<ACCOUNT_ID>.r2.cloudflarestorage.com

import { S3Client, PutObjectCommand, GetObjectCommand, ListObjectsV2Command } from '@aws-sdk/client-s3';
import { getSignedUrl } from '@aws-sdk/s3-request-presigner';
import dotenv from 'dotenv';
dotenv.config({ path: './.env.local' });

let s3Client;

const getS3Client = () => {
  if (s3Client) return s3Client;

  const endpoint = process.env.R2_ENDPOINT;
  if (!endpoint) throw new Error('R2_ENDPOINT is not set.');

  s3Client = new S3Client({
    region: 'auto',
    endpoint,
    credentials: {
      accessKeyId: process.env.R2_ACCESS_KEY_ID || '',
      secretAccessKey: process.env.R2_SECRET_ACCESS_KEY || '',
    },
    forcePathStyle: true, // required for MinIO and R2
  });

  return s3Client;
};

const getBucketName = () => {
  const bucket = process.env.R2_BUCKET_NAME;
  if (!bucket) throw new Error('R2_BUCKET_NAME is not set.');
  return bucket;
};

const buildPublicUrl = (key) => {
  const base = (process.env.R2_PUBLIC_URL || process.env.R2_ENDPOINT || '').replace(/\/$/, '');
  const bucket = getBucketName();
  // R2_PUBLIC_URL already includes the bucket (e.g. https://pub-xxx.r2.dev or http://localhost:9000/bucket)
  // If it ends with the bucket name, don't add it again.
  if (base.endsWith(`/${bucket}`)) {
    return `${base}/${key}`;
  }
  return `${base}/${bucket}/${key}`;
};

/**
 * Uploads a file buffer to R2 / MinIO.
 * @param {object} file - Multer file object (must have `buffer` and `mimetype`).
 * @param {string} destinationPath - Object key / path inside the bucket.
 * @returns {Promise<string>} Public URL of the uploaded file.
 */
export const uploadFileToStorage = async (file, destinationPath) => {
  if (!file || !file.buffer) throw new Error('No file buffer provided for upload.');
  if (!destinationPath) throw new Error('No destination path provided for upload.');

  await getS3Client().send(
    new PutObjectCommand({
      Bucket: getBucketName(),
      Key: destinationPath,
      Body: file.buffer,
      ContentType: file.mimetype,
    }),
  );

  return buildPublicUrl(destinationPath);
};

/**
 * Generates a presigned download URL for a private object.
 * @param {string} key - Object key / path inside the bucket.
 * @param {number} expiresIn - Expiry in seconds (default 3600).
 * @returns {Promise<string>} Presigned URL.
 */
export const getFileDownloadUrl = async (key, expiresIn = 3600) => {
  const command = new GetObjectCommand({ Bucket: getBucketName(), Key: key });
  return getSignedUrl(getS3Client(), command, { expiresIn });
};

/**
 * Lists all files under a given prefix.
 * @param {string} prefix - Folder prefix (e.g. 'books/covers/').
 * @returns {Promise<Array<{name: string, url: string}>>}
 */
export const listFilesInFolder = async (prefix) => {
  const response = await getS3Client().send(
    new ListObjectsV2Command({ Bucket: getBucketName(), Prefix: prefix }),
  );

  return (response.Contents || [])
    .filter((obj) => !obj.Key.endsWith('/'))
    .map((obj) => ({ name: obj.Key, url: buildPublicUrl(obj.Key) }));
};
