// Chart colours, read from the --rp-* tokens in index.css so the hex values
// live in one place. Read at render; the fallbacks are for tests (jsdom has
// no stylesheet).
const FALLBACK = {
  series1: '#c98500',
  series2: '#3987e5',
  grid: '#1b2338',
  axis: '#8b93ad',
  cursor: '#171f33',
};

const read = (name, fallback) => {
  if (typeof document === 'undefined') return fallback;
  return getComputedStyle(document.documentElement).getPropertyValue(name).trim() || fallback;
};

export const chartTheme = () => ({
  series1: read('--rp-series-1', FALLBACK.series1),
  series2: read('--rp-series-2', FALLBACK.series2),
  grid: read('--rp-hairline', FALLBACK.grid),
  axis: read('--rp-ink-holder', FALLBACK.axis),
  cursor: read('--rp-surface-2', FALLBACK.cursor),
});
