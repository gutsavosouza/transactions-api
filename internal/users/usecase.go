package users

import (
	"context"
	"errors"
	"log/slog"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	repo "github.com/gutsavosouza/transactions-api/internal/adapters/postgres/sqlc"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/paemuri/brdoc"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrCPFAlreadyExists = errors.New("cpf already in use")
	ErrInvalidCPF       = errors.New("invalid cpf provided")
	ErrInvalidInput     = errors.New("invalid input data")
)

type UseCase interface {
	NewUser(ctx context.Context, newUserData createUserDTO) (userResponse, error)
	FindByCPF(ctx context.Context, cpf string) (repo.User, error)
}

type uc struct {
	repo repo.Querier
}

func NewUseCase(repo repo.Querier) UseCase {
	return &uc{repo: repo}
}

func (uc *uc) NewUser(ctx context.Context, newUserDTO createUserDTO) (userResponse, error) {
	// validating data based off defined struct
	validator := validator.New()
	err := validator.Struct(newUserDTO)
	if err != nil {
		slog.Error("error validating data from body: %v", "error", err)
		return userResponse{}, ErrInvalidInput
	}

	// validating cpf
	isCPF := brdoc.IsCPF(newUserDTO.Cpf)
	if !isCPF {
		return userResponse{}, ErrInvalidCPF
	}

	//checking if user already exists
	existingUser, err := uc.FindByCPF(ctx, newUserDTO.Cpf)
	if err == nil && existingUser.ID.Valid {
		slog.Error("error while fetching user: %v", "error", err)
		return userResponse{}, ErrCPFAlreadyExists
	}

	hashedPass, err := bcrypt.GenerateFromPassword([]byte(newUserDTO.Password), bcrypt.DefaultCost)
	if err != nil {
		slog.Error("error hashing password from user: %v", "error", err)
		return userResponse{}, nil
	}

	user, err := uc.repo.CreateUser(ctx, repo.CreateUserParams{
		ID: pgtype.UUID{
			Bytes: uuid.New(),
			Valid: true,
		},
		Cpf:      newUserDTO.Cpf,
		Name:     newUserDTO.Name,
		Password: string(hashedPass),
	})
	if err != nil {
		slog.Error("error while creating the user: %v", "error", err)
		return userResponse{}, err
	}

	return userResponse{
		Cpf:  user.Cpf,
		Name: user.Name,
	}, nil
}

func (uc *uc) FindByCPF(ctx context.Context, cpf string) (repo.User, error) {
	user, err := uc.repo.FindUserByCPF(ctx, cpf)
	if err != nil {
		slog.Error("error while fetching user by cpf: %v", "error", err)
		return repo.User{}, err
	}

	return user, nil
}
