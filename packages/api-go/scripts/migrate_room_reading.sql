-- Migration: Room Detail (ROOM_DETAIL_2a-2) — a room reads either a standalone
-- book or a bucket (a shared reading list) with a current book chosen from it.
-- Run against your PostgreSQL database.

-- ============================================================
-- 1. Rooms can point at a USER bucket, not just a curated one
-- ============================================================
-- rooms.current_bucket_id was FK'd to curated_buckets only, so a room could
-- never read through one of the user's own buckets. Buckets live in two
-- tables (user_buckets / curated_buckets), so the reference is kept as a
-- plain id plus a type discriminator instead of a single-table FK.

ALTER TABLE rooms DROP CONSTRAINT IF EXISTS rooms_current_bucket_id_fkey;

ALTER TABLE rooms
    ADD COLUMN IF NOT EXISTS current_bucket_type VARCHAR(10);

ALTER TABLE rooms DROP CONSTRAINT IF EXISTS rooms_current_bucket_type_check;
ALTER TABLE rooms
    ADD CONSTRAINT rooms_current_bucket_type_check
    CHECK (current_bucket_type IS NULL OR current_bucket_type IN ('user', 'curated'));

-- A bucket reference is only meaningful with its type, and vice versa.
ALTER TABLE rooms DROP CONSTRAINT IF EXISTS rooms_bucket_pair_check;
ALTER TABLE rooms
    ADD CONSTRAINT rooms_bucket_pair_check
    CHECK ((current_bucket_id IS NULL) = (current_bucket_type IS NULL));

-- Existing rows pointed at curated buckets by definition of the old FK.
UPDATE rooms
   SET current_bucket_type = 'curated'
 WHERE current_bucket_id IS NOT NULL
   AND current_bucket_type IS NULL;

-- ============================================================
-- 2. The room creator is a member (they were never inserted)
-- ============================================================
INSERT INTO room_members (room_id, user_id, role, joined_at)
SELECT r.id, r.admin_id, 'admin', r.created_at
  FROM rooms r
 WHERE NOT EXISTS (
     SELECT 1 FROM room_members m
      WHERE m.room_id = r.id AND m.user_id = r.admin_id
 );

-- ============================================================
-- 3. Every room is joinable by code
-- ============================================================
-- Invite codes used to be generated for private rooms only, but the Room
-- Detail screen shows the code for every room ("Anyone with the code can
-- join this room"). Backfill is handled by the API on next create; existing
-- codeless rooms are listed here so they can be filled in deliberately.
SELECT id, name, is_private
  FROM rooms
 WHERE invite_code IS NULL;
