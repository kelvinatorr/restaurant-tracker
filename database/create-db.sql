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
    restaurant_id INTEGER NOT NULL REFERENCES restaurant(id) ON UPDATE CASCADE ON DELETE CASCADE, -- Must track the id in restaurant table
    visit_datetime TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', CURRENT_TIMESTAMP)), -- RFC3339 UTC timezone
    note TEXT,
    CHECK (length(visit_datetime) == 20)
);
CREATE INDEX IF NOT EXISTS visit_restaurant on visit (restaurant_id);

CREATE TABLE IF NOT EXISTS visit_user (
    id INTEGER PRIMARY KEY, -- Autoincrements per the documentation
    visit_id INTEGER NOT NULL REFERENCES visit(id) ON UPDATE CASCADE ON DELETE CASCADE, -- Must track the id in the visit table
    user_id INTEGER NOT NULL REFERENCES user(id) ON UPDATE CASCADE, -- Must track the id in user table
    rating INTEGER,
    CHECK ((rating > 0 and rating < 6) or rating is NULL)
);
CREATE INDEX IF NOT EXISTS visit_user_visit_id on visit_user (visit_id);
-- Can't have the same user more than once in the same visit.
CREATE UNIQUE INDEX IF NOT EXISTS visit_user_visit_id_user_id on visit_user (visit_id, user_id);

CREATE TABLE IF NOT EXISTS user (
    id INTEGER PRIMARY KEY, -- Autoincrements per the documentation
    first_name TEXT NOT NULL,
    last_name TEXT NOT NULL,
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT,
    remember_hash TEXT
);
CREATE UNIQUE INDEX IF NOT EXISTS remember_hash on user (remember_hash);

CREATE TABLE IF NOT EXISTS gmaps_place (
    id INTEGER PRIMARY KEY, -- Autoincrements per the documentation
    last_updated TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', CURRENT_TIMESTAMP)), -- RFC3339 UTC timezone
    place_id TEXT NOT NULL UNIQUE, -- Don't use this as the PK because it can change over time
    business_status TEXT,
    formatted_phone_number TEXT,
    name TEXT NOT NULL,
    price_level INTEGER,
    rating REAL,
    url TEXT, -- The url to this place Google Maps
    user_ratings_total INTEGER,
    utc_offset INTEGER, -- The number of minutes this place’s current timezone is offset from UTC
    website TEXT,
    restaurant_id INTEGER NOT NULL REFERENCES restaurant(id) ON UPDATE CASCADE ON DELETE CASCADE
);

