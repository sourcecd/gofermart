-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS security (
    id BIGSERIAL PRIMARY KEY,
    seckey VARCHAR(255)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE security;
-- +goose StatementEnd
