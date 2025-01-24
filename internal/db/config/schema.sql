CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_login TIMESTAMP
);

CREATE TABLE IF NOT EXISTS brewers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    location TEXT
);

CREATE TABLE IF NOT EXISTS beers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    brewer_id INTEGER,
    style TEXT,
    abv REAL NOT NULL,
    rating REAL,
    notes TEXT,
    FOREIGN KEY (brewer_id) REFERENCES brewers(id) ON DELETE SET NULL,
    CONSTRAINT unique_brewer_beer UNIQUE (name, brewer_id)
);