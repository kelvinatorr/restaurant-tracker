-- PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS city (
    id INTEGER PRIMARY KEY, -- Autoincrements per the documentation
    name TEXT NOT NULL,
    state TEXT NOT NULL,
    CHECK (length(state) == 2) -- Use ISO 3166-1 alpha-2 country code if not a US state
);
CREATE UNIQUE INDEX IF NOT EXISTS city_name_state on city (name, state);

CREATE TABLE IF NOT EXISTS restaurant (
    id INTEGER PRIMARY KEY, -- Autoincrements per the documentation
    name TEXT NOT NULL,
    cuisine TEXT NOT NULL,
    note TEXT,
    address TEXT,
    city_id INTEGER NOT NULL REFERENCES city(id) ON UPDATE CASCADE, -- Must track id in city table
    zipcode TEXT,
    latitude REAL,
    longitude REAL
);

CREATE TABLE IF NOT EXISTS visit (
    id INTEGER PRIMARY KEY, -- Autoincrements per the documentation
    restaurant_id INTEGER NOT NULL REFERENCES restaurant(id) ON UPDATE CASCADE, -- Must track the id in restaurant table
    user_id INTEGER NOT NULL REFERENCES user(id) ON UPDATE CASCADE, -- Must track the id in user table
    visit_datetime TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', CURRENT_TIMESTAMP)), -- RFC3339 UTC timezone
    note TEXT,
    rating INTEGER,
    CHECK ((rating > 0 and rating < 6) or rating is NULL)
    CHECK (length(visit_datetime) == 20)
);

CREATE TABLE IF NOT EXISTS user (
    id INTEGER PRIMARY KEY, -- Autoincrements per the documentation
    first_name TEXT NOT NULL,
    last_name TEXT NOT NULL,
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT
);
