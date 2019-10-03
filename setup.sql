CREATE TABLE IF NOT EXISTS issues(
    id BLOB NOT NULL PRIMARY KEY,
    title TEXT NOT NULL,
    description TEXT NOT NULL,
    createdAt TEXT,
    createdBy TEXT
);