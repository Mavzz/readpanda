-- Migration: Open rooms (specs/open-rooms, R7.1, R7.2).
-- Run against your PostgreSQL database before deploying the API that lists
-- open rooms.

-- Create room used to save is_private without anything reading it, and a
-- missing value saved as Open. No room should become findable without its
-- creator choosing it, so every existing room starts Invite only.
UPDATE rooms SET is_private = true WHERE is_private IS DISTINCT FROM true;

ALTER TABLE rooms ALTER COLUMN is_private SET DEFAULT true;
ALTER TABLE rooms ALTER COLUMN is_private SET NOT NULL;

-- GET /room/open filters on open rooms, optionally by the book they read.
CREATE INDEX IF NOT EXISTS rooms_open_idx ON rooms (current_book_id) WHERE NOT is_private;
