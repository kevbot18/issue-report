CREATE TABLE IF NOT EXISTS tickets(
    id BLOB NOT NULL PRIMARY KEY,
    title TEXT NOT NULL,
    description TEXT NOT NULL,
    createdAt TEXT,
    createdBy TEXT
);