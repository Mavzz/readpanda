import { useEffect, useRef, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { Banner, BookRow, Button, Card, Eyebrow, List, ListItem, PageHeader, Tag, TextAction, Toggle } from "../components/ui";
import { api } from "../services/api";
import { BOOKS_KEY } from "../services/queries";
import { readJSON } from "../utils/session";
import { parseCSVObjects, toCSV } from "../utils/csv";

// Bulk upload: pick one folder holding a books.csv manifest and the files it
// names. Each manifest row becomes one POST /books/upload, sent one at a time
// so a large batch doesn't hold every file in flight at once.
//
// books.csv columns: title, author, description, genre, subgenre, manuscript, cover
// `manuscript` and `cover` are paths relative to the chosen folder.

const COLUMNS = ["title", "author", "description", "genre", "subgenre", "manuscript", "cover"];
const MANUSCRIPT_TYPES = [".pdf", ".epub"];
const IMAGE_TYPES = [".jpg", ".jpeg", ".png", ".webp", ".gif"];
// Cloud Run rejects request bodies over 32 MiB, so these fail in production.
const REQUEST_LIMIT = 32 * 1024 * 1024;

const hasExt = (name, exts) => exts.some((ext) => name.toLowerCase().endsWith(ext));
// macOS may store names decomposed (é as e + ◌́); compare in one form.
const normPath = (p) => p.normalize("NFC").replace(/\\/g, "/").replace(/^\.?\//, "");
const formatSize = (bytes) => `${(bytes / 1048576).toFixed(1)} MB`;

const downloadTemplate = () => {
  const csv = toCSV([
    COLUMNS,
    ["The Last Wish", "Andrzej Sapkowski", "The first Witcher stories.", "Fantasy", "Dark Fantasy", "Witcher/The Last Wish.epub", "covers/the-last-wish.jpg"],
  ]);
  const url = URL.createObjectURL(new Blob([csv], { type: "text/csv" }));
  const a = Object.assign(document.createElement("a"), { href: url, download: "books.csv" });
  a.click();
  URL.revokeObjectURL(url);
};

// Turns manifest rows plus the folder's files into upload jobs, each with the
// problems that block it (errors) and the ones worth knowing (warnings).
const buildJobs = (rows, filesByPath) => {
  const genres = new Set(readJSON("genres", []).map((g) => g.value));
  const subgenres = readJSON("subgenres", []);

  return rows.map((row, index) => {
    const errors = [];
    const warnings = [];
    if (!row.title) errors.push("No title");
    if (!row.description) errors.push("No description");
    if (!row.genre) errors.push("No genre");

    const manuscript = row.manuscript ? filesByPath.get(normPath(row.manuscript)) : null;
    if (!row.manuscript) errors.push("No manuscript");
    else if (!manuscript) errors.push(`Manuscript not found: ${row.manuscript}`);
    else if (!hasExt(manuscript.name, MANUSCRIPT_TYPES)) errors.push("Manuscript isn't a PDF or EPUB");

    const cover = row.cover ? filesByPath.get(normPath(row.cover)) : null;
    if (row.cover && !cover) errors.push(`Cover not found: ${row.cover}`);
    else if (cover && !hasExt(cover.name, IMAGE_TYPES)) errors.push("Cover isn't an image");

    if (row.genre && genres.size && !genres.has(row.genre)) warnings.push(`"${row.genre}" isn't in the genre catalog`);
    if (row.subgenre && subgenres.length && !subgenres.some((s) => s.genre === row.genre && s.value === row.subgenre)) {
      warnings.push(`"${row.subgenre}" isn't a ${row.genre || "catalog"} subgenre`);
    }
    if (manuscript && manuscript.size + (cover?.size ?? 0) > REQUEST_LIMIT) {
      warnings.push(`${formatSize(manuscript.size)}: over the 32 MB production upload limit`);
    }

    return {
      key: `${index}-${row.manuscript}`,
      row,
      manuscript,
      cover,
      coverPreview: cover && !errors.length ? URL.createObjectURL(cover) : null,
      errors,
      warnings,
    };
  });
};

const STATUS_LABEL = { uploading: "Uploading…", done: "Uploaded", failed: "Failed" };

const BulkUploadPage = () => {
  const queryClient = useQueryClient();
  const folderInput = useRef(null);
  const [jobs, setJobs] = useState([]);
  const [folderName, setFolderName] = useState("");
  const [status, setStatus] = useState({}); // job key → { state, message }
  const [notifyReaders, setNotifyReaders] = useState(false);
  const [running, setRunning] = useState(null); // { at, of } while uploading
  const [notice, setNotice] = useState("");

  // Free cover previews when the job list is replaced or the page unmounts.
  useEffect(() => () => jobs.forEach((j) => j.coverPreview && URL.revokeObjectURL(j.coverPreview)), [jobs]);

  const pickFolder = async (fileList) => {
    const files = Array.from(fileList ?? []);
    if (!files.length) return;
    setNotice("");
    setStatus({});

    // webkitRelativePath is "<folder>/<path inside it>"; the manifest uses the latter.
    const root = files[0].webkitRelativePath.split("/")[0];
    const filesByPath = new Map(files.map((f) => [normPath(f.webkitRelativePath.slice(root.length + 1)), f]));
    const manifests = files.filter((f) => f.name.toLowerCase().endsWith(".csv") && !f.webkitRelativePath.slice(root.length + 1).includes("/"));
    const manifest = manifests.find((f) => f.name.toLowerCase() === "books.csv") ?? manifests[0];

    if (!manifest) {
      setJobs([]);
      setNotice(`"${root}" has no books.csv at its top level. Download the template to start one.`);
      return;
    }
    const rows = parseCSVObjects(await manifest.text());
    const missingColumns = ["title", "manuscript"].filter((c) => rows.length && !(c in rows[0]));
    if (!rows.length || missingColumns.length) {
      setJobs([]);
      setNotice(`${manifest.name} needs a header row with these columns: ${COLUMNS.join(", ")}.`);
      return;
    }
    setFolderName(`${root}/${manifest.name}`);
    setJobs(buildJobs(rows, filesByPath));
  };

  const pending = jobs.filter((j) => !j.errors.length && status[j.key]?.state !== "done");
  const blocked = jobs.filter((j) => j.errors.length).length;
  const done = jobs.filter((j) => status[j.key]?.state === "done").length;
  const failed = jobs.filter((j) => status[j.key]?.state === "failed").length;

  const uploadAll = async () => {
    setNotice("");
    let uploaded = 0;
    for (const [i, job] of pending.entries()) {
      setRunning({ at: i + 1, of: pending.length });
      setStatus((s) => ({ ...s, [job.key]: { state: "uploading" } }));
      try {
        const { row } = job;
        const data = new FormData();
        data.append("title", row.title);
        data.append("author_name", row.author ?? "");
        data.append("description", row.description);
        data.append("genre", row.genre);
        data.append("subgenre", row.subgenre ?? "");
        data.append("notify", String(notifyReaders));
        data.append("manuscript", job.manuscript);
        if (job.cover) data.append("cover", job.cover);
        await api.upload("/books/upload", data);
        uploaded++;
        setStatus((s) => ({ ...s, [job.key]: { state: "done" } }));
      } catch (err) {
        // A 413 comes from the platform, not the API, so it has no useful body.
        const message = err.status === 413 ? "Too large for the server" : err.message;
        setStatus((s) => ({ ...s, [job.key]: { state: "failed", message } }));
      }
    }
    setRunning(null);
    if (uploaded) queryClient.invalidateQueries({ queryKey: BOOKS_KEY });
    setNotice(`Uploaded ${uploaded} of ${pending.length} books.`);
  };

  const reset = () => {
    setJobs([]);
    setStatus({});
    setFolderName("");
    setNotice("");
    if (folderInput.current) folderInput.current.value = "";
  };

  return (
    <>
      <PageHeader title="Bulk upload" subtitle="Publish many books at once from a folder with a books.csv manifest">
        <Button variant="secondary" onClick={downloadTemplate}>Download template</Button>
      </PageHeader>

      <Banner message={notice} onDismiss={() => setNotice("")} />

      <Card className="p-6 flex flex-col gap-4">
        <div className="flex flex-wrap items-center justify-between gap-4">
          <div className="flex flex-col gap-1.5 min-w-0">
            <Eyebrow>Folder</Eyebrow>
            <span className="text-sm font-extrabold text-ink-title break-all">{folderName || "No folder chosen"}</span>
            <span className="text-xs font-semibold text-ink-holder">
              books.csv columns: {COLUMNS.join(", ")}. File paths are relative to the folder.
            </span>
          </div>
          <div className="flex items-center gap-3">
            {jobs.length > 0 && !running && <TextAction tone="muted" onClick={reset}>Clear</TextAction>}
            <Button variant="secondary" disabled={!!running} onClick={() => folderInput.current?.click()}>
              {jobs.length ? "Choose another folder" : "Choose folder"}
            </Button>
          </div>
        </div>
        <input ref={folderInput} type="file" webkitdirectory="" directory="" multiple hidden onChange={(e) => pickFolder(e.target.files)} />

        {jobs.length > 0 && (
          <div className="flex flex-wrap items-center justify-between gap-4">
            <label className="flex items-center gap-3 text-sm font-bold text-ink-body">
              <Toggle on={notifyReaders} label="Notify readers" onChange={setNotifyReaders} disabled={!!running} />
              Notify readers about each book
            </label>
            <div className="flex items-center gap-4">
              <span className="text-xs font-semibold text-ink-holder">
                {jobs.length} in manifest · {done} uploaded{failed ? ` · ${failed} failed` : ""}{blocked ? ` · ${blocked} need fixing` : ""}
              </span>
              <Button disabled={!!running || !pending.length} onClick={uploadAll}>
                {running ? `Uploading ${running.at} of ${running.of}…` : failed ? `Retry ${pending.length} books` : `Upload ${pending.length} books`}
              </Button>
            </div>
          </div>
        )}
      </Card>

      {jobs.length > 0 && (
        <List>
          {jobs.map((job) => {
            const { row } = job;
            const st = status[job.key];
            const meta = [row.author, [row.genre, row.subgenre].filter(Boolean).join(" / "), job.manuscript && formatSize(job.manuscript.size)]
              .filter(Boolean).join(" · ");
            const tags = [
              ...job.errors.map((e) => <Tag key={e} dot>{e}</Tag>),
              ...job.warnings.map((w) => <Tag key={w} muted>{w}</Tag>),
              st?.state === "failed" && st.message && <Tag key="err" dot>{st.message}</Tag>,
            ].filter(Boolean);
            return (
              <ListItem key={job.key}>
                <BookRow
                  cover={job.coverPreview}
                  title={row.title || "(no title)"}
                  meta={meta}
                  tags={tags.length ? tags : null}
                  trailing={
                    <span className={`text-xs font-extrabold whitespace-nowrap ${st?.state === "done" ? "text-link" : "text-ink-holder"}`}>
                      {job.errors.length ? "Skipped" : STATUS_LABEL[st?.state] ?? "Ready"}
                    </span>
                  }
                />
              </ListItem>
            );
          })}
        </List>
      )}
    </>
  );
};

export default BulkUploadPage;
