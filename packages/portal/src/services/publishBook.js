import { api } from './api';

// Publishing a book takes three requests, so files of any size skip the API
// (Cloud Run rejects request bodies over 32 MiB):
//   1. POST /books/upload-urls  → a book id and a presigned URL per file
//   2. PUT each file straight to storage
//   3. POST /books/upload       → the book's details and the stored keys

export const MANUSCRIPT_LIMIT = 500 * 1024 * 1024;
export const COVER_LIMIT = 20 * 1024 * 1024;
export const COVER_TYPES = ['.jpg', '.jpeg', '.png', '.webp', '.gif'];

// fetch can't report upload progress, so the storage PUT uses XHR.
const putFile = (target, file, onProgress) =>
  new Promise((resolve, reject) => {
    const xhr = new XMLHttpRequest();
    xhr.open('PUT', target.url);
    // Storage keeps this as the file's type, which readers download with.
    xhr.setRequestHeader('Content-Type', target.content_type);
    xhr.upload.onprogress = (e) => onProgress(e.loaded);
    xhr.onload = () =>
      xhr.status >= 200 && xhr.status < 300
        ? resolve()
        : reject(new Error(`Storage rejected the ${file.name} upload (${xhr.status})`));
    // A CORS rejection from the bucket lands here too, with no status.
    xhr.onerror = () => reject(new Error(`Couldn't reach storage to upload ${file.name}`));
    xhr.send(file);
  });

/**
 * @param {object} details  title, author_name, description, genre, subgenre, notify
 * @param {File} manuscript
 * @param {File} [cover]
 * @param {(fraction: number) => void} [onProgress]  0..1 over both files
 */
export const publishBook = async (details, manuscript, cover, onProgress = () => {}) => {
  const describe = (f) => f && { name: f.name, size: f.size };
  const targets = await api.post('/books/upload-urls', {
    manuscript: describe(manuscript),
    cover: describe(cover),
  });

  const total = manuscript.size + (cover?.size ?? 0);
  let coverSent = 0;
  if (cover) {
    await putFile(targets.cover, cover, (sent) => {
      coverSent = sent;
      onProgress(sent / total);
    });
    coverSent = cover.size;
  }
  await putFile(targets.manuscript, manuscript, (sent) => onProgress((coverSent + sent) / total));

  return api.post('/books/upload', {
    ...details,
    book_id: targets.book_id,
    manuscript_key: targets.manuscript.key,
    cover_key: targets.cover?.key ?? '',
  });
};
