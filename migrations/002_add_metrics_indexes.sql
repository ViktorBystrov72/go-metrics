-- +goose Up
-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS idx_metrics_name_type ON metrics(name, type);
CREATE INDEX IF NOT EXISTS idx_metrics_created_at ON metrics(created_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_metrics_name_type;
DROP INDEX IF EXISTS idx_metrics_created_at;
-- +goose StatementEnd 