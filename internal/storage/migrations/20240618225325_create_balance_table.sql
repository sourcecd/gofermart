-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS balance (
    userid BIGINT PRIMARY KEY,
    current DOUBLE PRECISION CHECK (current >= 0),
    withdrawn DOUBLE PRECISION
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE balance;
-- +goose StatementEnd
