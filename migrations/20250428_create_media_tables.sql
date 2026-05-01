-- Tabla para almacenar archivos multimedia (imágenes, videos)
CREATE TABLE IF NOT EXISTS media_files (
    id SERIAL PRIMARY KEY,
    key VARCHAR(255) UNIQUE NOT NULL,           -- ej: "video1.mp4", "AndresMelo.png"
    mime_type VARCHAR(100) NOT NULL,            -- ej: "video/mp4", "image/png"
    data BYTEA NOT NULL,                         -- contenido binario
    size_bytes INTEGER NOT NULL,                 -- tamaño en bytes
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_media_files_key ON media_files(key);

-- Tabla para metadata adicional (opcional)
CREATE TABLE IF NOT EXISTS media_metadata (
    id SERIAL PRIMARY KEY,
    media_id INTEGER REFERENCES media_files(id) ON DELETE CASCADE,
    width INTEGER,                               -- ancho en píxeles (para imágenes/videos)
    height INTEGER,                              -- alto en píxeles
    duration_seconds FLOAT,                      -- duración en segundos (para videos)
    alt_text VARCHAR(500),                       -- texto alternativo
    created_at TIMESTAMP DEFAULT NOW()
);
