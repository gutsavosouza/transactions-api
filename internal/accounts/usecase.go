package accounts

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"github.com/google/uuid"
	repo "github.com/gutsavosouza/transactions-api/internal/adapters/postgres/sqlc"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"
)

var (
	ErrAccountNotFound     = errors.New("account not found")
	ErrUnauthorizedAccount = errors.New("user does not own this account")
)

type UseCase interface {
	CreateAccount(ctx context.Context, userID pgtype.UUID) (accountResponseDTO, error)
	GetAccount(ctx context.Context, accountID pgtype.UUID) (accountResponseDTO, error)
	GetAllAccounts(ctx context.Context, userID pgtype.UUID, limit, offset int32) ([]accountResponseDTO, error)
	AddBalance(ctx context.Context, userID pgtype.UUID, accountID pgtype.UUID, amount pgtype.Numeric) (accountResponseDTO, error)
}

type uc struct {
	repo repo.Querier
	mu   sync.Mutex
}

func NewUseCase(repo repo.Querier) UseCase {
	return &uc{repo: repo}
}

func (uc *uc) CreateAccount(ctx context.Context, userID pgtype.UUID) (accountResponseDTO, error) {
	account, err := uc.repo.CreateAccount(ctx, repo.CreateAccountParams{
		ID: pgtype.UUID{
			Bytes: uuid.New(),
			Valid: true,
		},
		UserID: userID,
	})
	if err != nil {
		slog.Error("error while creating account: %v", "error", err)
		return accountResponseDTO{}, fmt.Errorf("error while creating account entry")
	}

	return accountResponseDTO{
		ID:        account.ID,
		UserID:    account.UserID,
		Balance:   account.Balance,
		CreatedAt: account.CreatedAt,
		UpdatedAt: account.UpdatedAt,
	}, nil
}

func (uc *uc) GetAccount(ctx context.Context, accountID pgtype.UUID) (accountResponseDTO, error) {
	account, err := uc.repo.GetAccount(ctx, accountID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return accountResponseDTO{}, ErrAccountNotFound
		}
		slog.Error("error retrieving account from id", "error", err)
		return accountResponseDTO{}, fmt.Errorf("error while retrieving account")
	}

	return accountResponseDTO{
		ID:        account.ID,
		UserID:    account.UserID,
		Balance:   account.Balance,
		CreatedAt: account.CreatedAt,
		UpdatedAt: account.UpdatedAt,
	}, nil
}

func (uc *uc) GetAllAccounts(ctx context.Context, userID pgtype.UUID, limit, offset int32) ([]accountResponseDTO, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	accounts, err := uc.repo.ListUserAccounts(ctx, repo.ListUserAccountsParams{
		UserID: userID,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		slog.Error("error fetching all account for user: %v", "error", err)
		return []accountResponseDTO{}, nil
	}

	accountDTOs := make([]accountResponseDTO, len(accounts))
	for i, account := range accounts {
		accountDTOs[i] = accountResponseDTO{
			ID:        account.ID,
			UserID:    account.UserID,
			Balance:   account.Balance,
			CreatedAt: account.CreatedAt,
			UpdatedAt: account.UpdatedAt,
		}
	}

	return accountDTOs, nil
}

func (uc *uc) AddBalance(ctx context.Context, userID pgtype.UUID, accountID pgtype.UUID, amount pgtype.Numeric) (accountResponseDTO, error) {
	// lock to prevent concurrent balance updates
	uc.mu.Lock()
	defer uc.mu.Unlock()

	account, err := uc.repo.GetAccount(ctx, accountID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return accountResponseDTO{}, ErrAccountNotFound
		}
		slog.Error("error retrieving account", "error", err)
		return accountResponseDTO{}, fmt.Errorf("error retrieving account")
	}

	if account.UserID.Bytes != userID.Bytes {
		return accountResponseDTO{}, ErrUnauthorizedAccount
	}

	currentBalance := decimal.NewFromBigInt(account.Balance.Int, account.Balance.Exp)
	amountDecimal := decimal.NewFromBigInt(amount.Int, amount.Exp)
	newBalance := currentBalance.Add(amountDecimal)

	updatedAccount, err := uc.repo.UpdateBalance(ctx, repo.UpdateBalanceParams{
		ID: accountID,
		Balance: pgtype.Numeric{
			Int:   newBalance.Coefficient(),
			Exp:   newBalance.Exponent(),
			Valid: true,
		},
	})
	if err != nil {
		slog.Error("error updating account balance", "error", err)
		return accountResponseDTO{}, fmt.Errorf("error updating balance")
	}

	return accountResponseDTO{
		ID:        updatedAccount.ID,
		UserID:    updatedAccount.UserID,
		Balance:   updatedAccount.Balance,
		CreatedAt: updatedAccount.CreatedAt,
		UpdatedAt: updatedAccount.UpdatedAt,
	}, nil
}
