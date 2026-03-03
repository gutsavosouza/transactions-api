-- +goose Up
CREATE TYPE transaction_status AS ENUM('pending','completed', 'failed');

CREATE TABLE IF NOT EXISTS transactions (
    id UUID PRIMARY KEY,
    user_id UUID REFERENCES users(id) NOT NULL,
    from_account_id UUID REFERENCES accounts(id) NOT NULL,
    to_account_id UUID REFERENCES accounts(id) NOT NULL,
    amount NUMERIC(19, 2) NOT NULL,
    status transaction_status NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL
);

-- +goose Down
DROP TABLE IF EXISTS transactions;
DROP TYPE IF EXISTS transaction_status;