package authentication

import (
	"context"
	"log/slog"
	"time"

	repo "github.com/gutsavosouza/transactions-api/internal/adapters/postgres/sqlc"
	"github.com/gutsavosouza/transactions-api/internal/token"
	"golang.org/x/crypto/bcrypt"
)

type UseCase interface {
	Login(ctx context.Context, loginDTO LoginDTO) (LoginResponseDTO, error)
	GetTokenMaker() *token.JWTMaker
}

type uc struct {
	repo       repo.Querier
	TokenMaker *token.JWTMaker
}

func NewUseCase(repo repo.Querier, secretKey string) UseCase {
	return &uc{
		repo:       repo,
		TokenMaker: token.NewJWTMaker(secretKey),
	}
}

func (uc *uc) Login(ctx context.Context, loginDTO LoginDTO) (LoginResponseDTO, error) {
	existingUser, err := uc.repo.FindUserByCPF(ctx, loginDTO.Cpf)
	if err != nil {
		slog.Error("error while fetching user: %v", "error", err)
		return LoginResponseDTO{}, err
	}

	err = bcrypt.CompareHashAndPassword([]byte(existingUser.Password), []byte(loginDTO.Password))
	if err != nil {
		slog.Error("error while checking password: %v", "error", err)
		return LoginResponseDTO{}, err
	}

	tokenString, _, err := uc.TokenMaker.CreateToken(existingUser.ID, existingUser.Cpf, existingUser.Name, time.Hour*1)
	if err != nil {
		slog.Error("error while generating acess token with JWT: %v", "error", err)
		return LoginResponseDTO{}, err
	}

	return LoginResponseDTO{
		AccessToken: tokenString,
	}, nil
}

func (uc *uc) GetTokenMaker() *token.JWTMaker {
	return uc.TokenMaker
}
