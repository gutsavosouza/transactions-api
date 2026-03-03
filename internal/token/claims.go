package token

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type UserClaims struct {
	UserID pgtype.UUID `json:"id"`
	Cpf    string      `json:"cpf"`
	Name   string      `json:"name"`
	jwt.RegisteredClaims
}

func NewUserClaims(id pgtype.UUID, cpf string, name string, duration time.Duration) (*UserClaims, error) {
	tokenId, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}

	return &UserClaims{
		UserID: id,
		Cpf:    cpf,
		Name:   name,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        tokenId.String(),
			Subject:   cpf,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(duration)),
		},
	}, nil
}
