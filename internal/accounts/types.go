package accounts

import "github.com/jackc/pgx/v5/pgtype"

type accountResponseDTO struct {
	ID        pgtype.UUID        `json:"id"`
	UserID    pgtype.UUID        `json:"user_id"`
	Balance   pgtype.Numeric     `json:"balance"`
	CreatedAt pgtype.Timestamptz `json:"created_at"`
	UpdatedAt pgtype.Timestamptz `json:"updated_at"`
}
