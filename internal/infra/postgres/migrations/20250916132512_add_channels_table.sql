-- +goose Up
CREATE TABLE IF NOT EXISTS channels
(
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    creator_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    is_public BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP

    FOREIGN KEY (creator_id) REFERENCES users(id)
);

-- +goose Down
DROP TABLE IF EXISTS channels;
