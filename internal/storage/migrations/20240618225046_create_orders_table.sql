-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS orders (
    userid BIGINT,
    number BIGINT PRIMARY KEY,
    uploaded_at TIMESTAMPTZ,
    status VARCHAR(255),
    accrual DOUBLE PRECISION,
    sum DOUBLE PRECISION,
    processed_at TIMESTAMPTZ,
    processable bool,
    processed bool
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE orders;
-- +goose StatementEnd
