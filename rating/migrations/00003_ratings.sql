-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS ratings (
    record_id VARCHAR(255) NOT NULL,
    record_type VARCHAR(255) NOT NULL,
    user_id VARCHAR(255) REFERENCES users(id),
    value INT NOT NULL,
    PRIMARY KEY (record_id, record_type, user_id)
);

-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS ratings;

-- +goose StatementEnd
