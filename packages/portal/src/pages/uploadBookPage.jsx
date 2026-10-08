import { useEffect, useRef, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { Banner, Button, Card, Cover, Eyebrow, Field, PageHeader } from "../components/ui";
import { UploadIcon } from "../components/icons";
import { COVER_LIMIT, COVER_TYPES, MANUSCRIPT_LIMIT, publishBook } from "../services/publishBook";
import { BOOKS_KEY } from "../services/queries";
import { readJSON } from "../utils/session";

const MANUSCRIPT_TYPES = [".pdf", ".epub"];
const hasExt = (file, exts) => exts.some((ext) => file.name.toLowerCase().endsWith(ext));

const EMPTY = { title: "", author: "", description: "", genre: "", subgenre: "" };

const UploadBookPage = () => {
  const queryClient = useQueryClient();
  const [form, setForm] = useState(EMPTY);
  const [coverFile, setCoverFile] = useState(null);
  const [coverPreview, setCoverPreview] = useState(null);
  const [manuscript, setManuscript] = useState(null);
  const [dragging, setDragging] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [progress, setProgress] = useState(0);
  const [notice, setNotice] = useState("");
  const coverInput = useRef(null);
  const manuscriptInput = useRef(null);

  const genres = readJSON("genres", []);
  const subgenres = readJSON("subgenres", []).filter((s) => s.genre === form.genre);

  // Free the preview's object URL when it's replaced or the page unmounts.
  useEffect(() => () => { if (coverPreview) URL.revokeObjectURL(coverPreview); }, [coverPreview]);

  const set = (key) => (e) => setForm((f) => ({ ...f, [key]: e.target.value }));
  // A subgenre only makes sense under the genre it was picked for.
  const setGenre = (e) => setForm((f) => ({ ...f, genre: e.target.value, subgenre: "" }));

  const pickCover = (file) => {
    if (file && !hasExt(file, COVER_TYPES)) {
      setNotice("The cover has to be a JPEG, PNG, WebP or GIF image.");
      return;
    }
    if (file && file.size > COVER_LIMIT) {
      setNotice(`The cover is over the ${COVER_LIMIT >> 20} MB limit.`);
      return;
    }
    setCoverFile(file ?? null);
    setCoverPreview(file ? URL.createObjectURL(file) : null);
  };

  const pickManuscript = (file) => {
    if (file && !hasExt(file, MANUSCRIPT_TYPES)) {
      setNotice("The manuscript has to be a PDF or EPUB.");
      return;
    }
    if (file && file.size > MANUSCRIPT_LIMIT) {
      setNotice(`The manuscript is over the ${MANUSCRIPT_LIMIT >> 20} MB limit.`);
      return;
    }
    setManuscript(file ?? null);
  };

  const onDrop = (e) => {
    e.preventDefault();
    setDragging(false);
    pickManuscript(e.dataTransfer.files?.[0]);
  };

  const missing = [
    !form.title.trim() && "a title",
    !form.description.trim() && "a description",
    !form.genre && "a genre",
    !manuscript && "a manuscript",
  ].filter(Boolean);
  const ready = missing.length === 0 && !uploading;

  const publish = async (e) => {
    e.preventDefault();
    if (!ready) return;
    setNotice("");
    setUploading(true);
    setProgress(0);
    try {
      const details = {
        title: form.title.trim(),
        author_name: form.author.trim(),
        description: form.description.trim(),
        genre: form.genre,
        subgenre: form.subgenre,
      };
      await publishBook(details, manuscript, coverFile, setProgress);
      queryClient.invalidateQueries({ queryKey: BOOKS_KEY });
      setNotice(`"${form.title.trim()}" is published. Readers get a notification.`);
      setForm(EMPTY);
      pickCover(null);
      setManuscript(null);
    } catch (err) {
      setNotice(`Couldn't publish the book: ${err.message}`);
    } finally {
      setUploading(false);
    }
  };

  return (
    <>
      <PageHeader title="Upload book" subtitle="Published books show up for every reader right away" />

      <Banner message={notice} onDismiss={() => setNotice("")} />

      <form onSubmit={publish} className="grid grid-cols-1 lg:grid-cols-[minmax(0,1fr)_340px] gap-4 items-start">
        <Card className="p-6 flex flex-col gap-4">
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <Field label="Title" value={form.title} onChange={set("title")} />
            <Field label="Author" value={form.author} onChange={set("author")} placeholder="Optional" />
          </div>
          <Field label="Description" multiline rows={6} value={form.description} onChange={set("description")} />
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <Field label="Genre" as="select" value={form.genre} onChange={setGenre}>
              <option value="">Pick a genre</option>
              {genres.map((g) => <option key={g.value} value={g.value}>{g.label}</option>)}
            </Field>
            <Field label="Subgenre" as="select" value={form.subgenre} onChange={set("subgenre")} disabled={!form.genre}>
              <option value="">{form.genre ? "Pick a subgenre" : "Pick a genre first"}</option>
              {subgenres.map((s) => <option key={s.value} value={s.value}>{s.label}</option>)}
            </Field>
          </div>
        </Card>

        <div className="flex flex-col gap-4">
          <Card className="p-6 flex flex-col gap-3">
            <Eyebrow>Cover</Eyebrow>
            <div className="flex items-end gap-4">
              <Cover src={coverPreview} title={form.title || "cover"} className="w-28 h-[168px] rounded-cover" />
              <div className="flex flex-col gap-2">
                <Button variant="secondary" onClick={() => coverInput.current?.click()}>
                  {coverFile ? "Change image" : "Choose image"}
                </Button>
                <span className="text-[11px] font-semibold text-ink-holder">Optional · 2:3 looks best</span>
              </div>
            </div>
            <input ref={coverInput} type="file" accept={COVER_TYPES.join(",")} hidden onChange={(e) => pickCover(e.target.files?.[0])} />
          </Card>

          <Card className="p-6 flex flex-col gap-3">
            <Eyebrow>Manuscript</Eyebrow>
            <button
              type="button"
              onClick={() => manuscriptInput.current?.click()}
              onDragOver={(e) => { e.preventDefault(); setDragging(true); }}
              onDragLeave={() => setDragging(false)}
              onDrop={onDrop}
              className={`rounded-card px-4 py-7 flex flex-col items-center gap-2 text-center shadow-[inset_0_0_0_1.5px_var(--rp-dashed)] ${dragging ? "bg-surface-2" : "bg-transparent hover:bg-surface-2"}`}
            >
              <span className="w-[38px] h-[38px] rounded-full bg-surface-2 flex items-center justify-center text-link">
                <UploadIcon width={18} height={18} />
              </span>
              <span className="text-sm font-extrabold text-ink-title break-all">{manuscript ? manuscript.name : "Drop a PDF or EPUB"}</span>
              <span className="text-xs font-semibold text-ink-holder">{manuscript ? "Click to replace" : `or click to choose · up to ${MANUSCRIPT_LIMIT >> 20} MB`}</span>
            </button>
            <input ref={manuscriptInput} type="file" accept={MANUSCRIPT_TYPES.join(",")} hidden onChange={(e) => pickManuscript(e.target.files?.[0])} />
          </Card>

          <Button type="submit" disabled={!ready} className="w-full">
            {uploading ? `Uploading ${Math.round(progress * 100)}%…` : "Publish book"}
          </Button>
          {!uploading && missing.length > 0 && (
            <span className="text-xs font-semibold text-ink-holder text-center">Still needs {missing.length > 1 ? `${missing.slice(0, -1).join(", ")} and ${missing.at(-1)}` : missing[0]}.</span>
          )}
        </div>
      </form>
    </>
  );
};

export default UploadBookPage;
