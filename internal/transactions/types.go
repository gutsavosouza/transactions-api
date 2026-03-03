package transactions

import (
	repo "github.com/gutsavosouza/transactions-api/internal/adapters/postgres/sqlc"
	"github.com/jackc/pgx/v5/pgtype"
)

type createTransactionDTO struct {
	FromAccountID string  `json:"from_account_id"`
	ToAccountID   string  `json:"to_account_id"`
	Amount        float64 `json:"amount"`
}

type transactionResponseDTO struct {
	ID            pgtype.UUID            `json:"id"`
	UserID        pgtype.UUID            `json:"user_id"`
	FromAccountID pgtype.UUID            `json:"from_account_id"`
	ToAccountID   pgtype.UUID            `json:"to_account_id"`
	Amount        pgtype.Numeric         `json:"amount"`
	Status        repo.TransactionStatus `json:"status"`
	CreatedAt     pgtype.Timestamptz     `json:"created_at"`
	UpdatedAt     pgtype.Timestamptz     `json:"updated_at"`
}
