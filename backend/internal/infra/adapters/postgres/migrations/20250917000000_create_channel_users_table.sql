-- +goose Up
CREATE TABLE IF NOT EXISTS channel_users (
    user_id UUID NOT NULL,
    channel_id UUID NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, channel_id),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (channel_id) REFERENCES channels(id) ON DELETE CASCADE
);

-- +goose Down
DROP TABLE IF EXISTS channel_users;
