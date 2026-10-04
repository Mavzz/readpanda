// Minimal RFC 4180 CSV: quoted fields may hold commas, quotes ("") and
// newlines. Good enough for a hand-edited or spreadsheet-exported manifest.

export const parseCSV = (text) => {
  const rows = [];
  let row = [];
  let field = '';
  let quoted = false;
  const src = text.replace(/^\uFEFF/, ''); // Excel adds a BOM

  for (let i = 0; i < src.length; i++) {
    const ch = src[i];
    if (quoted) {
      if (ch === '"' && src[i + 1] === '"') { field += '"'; i++; }
      else if (ch === '"') quoted = false;
      else field += ch;
    } else if (ch === '"') {
      quoted = true;
    } else if (ch === ',') {
      row.push(field); field = '';
    } else if (ch === '\n' || ch === '\r') {
      if (ch === '\r' && src[i + 1] === '\n') i++;
      row.push(field); field = '';
      rows.push(row); row = [];
    } else {
      field += ch;
    }
  }
  if (field !== '' || row.length) { row.push(field); rows.push(row); }
  return rows.filter((r) => r.some((cell) => cell.trim() !== ''));
};

// Rows as objects keyed by the lower-cased header row.
export const parseCSVObjects = (text) => {
  const [header = [], ...rows] = parseCSV(text);
  const keys = header.map((h) => h.trim().toLowerCase());
  return rows.map((r) => Object.fromEntries(keys.map((k, i) => [k, (r[i] ?? '').trim()])));
};

const quote = (v) => (/[",\r\n]/.test(v) ? `"${v.replace(/"/g, '""')}"` : v);
export const toCSV = (rows) => rows.map((r) => r.map((v) => quote(String(v ?? ''))).join(',')).join('\n') + '\n';
