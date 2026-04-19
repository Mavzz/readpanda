-- Migration: Create curated buckets and user buckets tables
-- Run against your PostgreSQL database

-- ============================================================
-- Story 1: Curated "Our Picks" Buckets
-- ============================================================

CREATE TABLE IF NOT EXISTS curated_buckets (
    id VARCHAR(50) PRIMARY KEY,
    title VARCHAR(100) NOT NULL,
    sort_order INT DEFAULT 0,
    cover_image_url TEXT,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS curated_bucket_books (
    bucket_id VARCHAR(50) REFERENCES curated_buckets(id) ON DELETE CASCADE,
    book_id INT REFERENCES books(book_id),
    sort_order INT DEFAULT 0,
    added_at TIMESTAMP DEFAULT NOW(),
    PRIMARY KEY (bucket_id, book_id)
);

-- ============================================================
-- Story 2: User Buckets
-- ============================================================

CREATE TABLE IF NOT EXISTS user_buckets (
    id VARCHAR(50) PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users(uuid) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_user_buckets_user_id ON user_buckets(user_id);

CREATE TABLE IF NOT EXISTS user_bucket_books (
    bucket_id VARCHAR(50) REFERENCES user_buckets(id) ON DELETE CASCADE,
    book_id INT REFERENCES books(book_id),
    added_at TIMESTAMP DEFAULT NOW(),
    PRIMARY KEY (bucket_id, book_id)
);
