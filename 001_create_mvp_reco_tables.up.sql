-- Tabla de metadata de libros recomendados (Google Books), para que el
-- "te puede interesar" muestre título, autor y portada.
-- ADITIVA: no toca la tabla 'recommendations' de Diego.

CREATE TABLE IF NOT EXISTS recommended_books (
    book_id          UUID         PRIMARY KEY,
    google_volume_id VARCHAR(64)  NOT NULL,
    title            TEXT         NOT NULL,
    authors          TEXT[]       NOT NULL DEFAULT '{}',
    thumbnail        TEXT,
    info_link        TEXT,
    description      TEXT,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_recommended_books_volume
    ON recommended_books(google_volume_id);
