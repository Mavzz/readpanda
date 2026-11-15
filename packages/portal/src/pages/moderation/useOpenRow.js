// A row on the Readers or Rooms tab opens the queue filtered to it, as a
// new history entry, so back returns to the list (portal-moderation R4.4,
// R5.3, R6.2). Clicks on the row's own buttons and links are left alone.
export const useOpenRow = (update, aboutKey) => (id) => (e) => {
  if (e?.target.closest?.('a, button')) return;
  update({ tab: 'queue', status: 'all', about_user: null, about_room: null, [aboutKey]: id, filter: null, sort: null }, { push: true });
};

// The same place as a link, for keyboards and new tabs.
export const queueLink = (aboutKey, id) => `/reports?tab=queue&status=all&${aboutKey}=${encodeURIComponent(id)}`;
