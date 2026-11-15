const round1 = (n) => String(Math.round(n * 10) / 10);

// Hours as the unit a person would say (R7.8).
export const formatHours = (h) => {
  if (h == null) return '–';
  if (h < 1) return `${Math.max(1, Math.round(h * 60))} min`;
  if (h < 48) return `${round1(h)} h`;
  return `${round1(h / 24)} days`;
};
