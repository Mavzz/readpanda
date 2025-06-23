CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(100) UNIQUE NOT NULL,
    password VARCHAR(300) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE user_preferences (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    preferences JSONB[] NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id)
);

--books table used to store the metadata for the books in the database the actual book is stored in firebase storage 
CREATE TABLE books (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    genre VARCHAR(50),
    cover_image_url TEXT NOT NULL,       -- URL for the cover image in Firebase Storage
    manuscript_url TEXT NOT NULL,        -- URL for the book manuscript in Firebase Storage
    status VARCHAR(50) DEFAULT 'Draft',  -- e.g., 'Draft', 'Live', 'Archived' (matches mockBooks status)
    views INTEGER DEFAULT 0,             -- Initial views count (matches mockBooks views)
    earnings DECIMAL(10, 2) DEFAULT 0.00,-- Initial earnings (matches mockBooks earnings)
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);