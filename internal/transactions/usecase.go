package transactions

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	repo "github.com/gutsavosouza/transactions-api/internal/adapters/postgres/sqlc"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"
)

var (
	ErrInvalidAmount       = errors.New("invalid amount")
	ErrAccountNotFound     = errors.New("account not found")
	ErrInsufficientBalance = errors.New("insufficient balance")
	ErrSameAccount         = errors.New("from and to accounts cannot be the same")
	ErrUnauthorizedAccount = errors.New("user does not own the from account")
)

type UseCase interface {
	CreateTransaction(ctx context.Context, userID pgtype.UUID, fromAccountID, toAccountID pgtype.UUID, amount pgtype.Numeric) (transactionResponseDTO, error)
	GetUserTransactions(ctx context.Context, userID pgtype.UUID, fromAccountID, toAccountID *pgtype.UUID, limit, offset int32) ([]transactionResponseDTO, error)
}

type uc struct {
	repo repo.Querier
	mu   sync.Mutex
}

func NewUseCase(repo repo.Querier) UseCase {
	return &uc{
		repo: repo,
	}
}

func (uc *uc) CreateTransaction(ctx context.Context, userID pgtype.UUID, fromAccountID, toAccountID pgtype.UUID, amount pgtype.Numeric) (transactionResponseDTO, error) {
	// validating accounts ids are different
	if fromAccountID.Bytes == toAccountID.Bytes {
		return transactionResponseDTO{}, ErrSameAccount
	}

	// validanting amount to be positive
	amountDecimal := decimal.NewFromBigInt(amount.Int, amount.Exp)
	if amountDecimal.LessThanOrEqual(decimal.Zero) {
		return transactionResponseDTO{}, ErrInvalidAmount
	}

	// retrieving from account to validate the ownership of the user
	fromAccount, err := uc.repo.GetAccount(ctx, fromAccountID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return transactionResponseDTO{}, ErrAccountNotFound
		}
		slog.Error("error retrieving from account", "error", err)
		return transactionResponseDTO{}, fmt.Errorf("error retrieving from account")
	}
	// ownership
	if fromAccount.UserID.Bytes != userID.Bytes {
		return transactionResponseDTO{}, ErrUnauthorizedAccount
	}

	// retrieving to account to validate it does exists
	_, err = uc.repo.GetAccount(ctx, toAccountID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return transactionResponseDTO{}, ErrAccountNotFound
		}
		slog.Error("error retrieving to account", "error", err)
		return transactionResponseDTO{}, fmt.Errorf("error retrieving to account")
	}

	// checking for suficient funds before initializing the transaction, and will check again in the process of validting the transaction just to be sure
	fromBalance := decimal.NewFromBigInt(fromAccount.Balance.Int, fromAccount.Balance.Exp)

	if fromBalance.LessThan(amountDecimal) {
		return transactionResponseDTO{}, ErrInsufficientBalance
	}

	// creating transactions with pendind status
	transaction, err := uc.repo.CreateTransaction(ctx, repo.CreateTransactionParams{
		ID: pgtype.UUID{
			Bytes: uuid.New(),
			Valid: true,
		},
		UserID:        userID,
		FromAccountID: fromAccountID,
		ToAccountID:   toAccountID,
		Amount:        amount,
		Status:        repo.TransactionStatusPending,
	})
	if err != nil {
		slog.Error("error creating transaction", "error", err)
		return transactionResponseDTO{}, fmt.Errorf("error creating transaction")
	}

	// initialziing a new gorouting to process the transaction properly
	go uc.processTransaction(context.Background(), transaction.ID, fromAccountID, toAccountID, amountDecimal)

	// sleeping a little in case the transaction does fisnish quickly
	time.Sleep(800 * time.Millisecond)
	// could be better to implem,ent a channel and listen to that when the transaction is done but this is a simpler
	updatedTransaction, err := uc.repo.GetTransaction(ctx, transaction.ID)
	if err != nil {
		slog.Warn("error fetching updated transaction, returning original", "error", err, "transaction_id", transaction.ID)
		return transactionResponseDTO{
			ID:            transaction.ID,
			UserID:        transaction.UserID,
			FromAccountID: transaction.FromAccountID,
			ToAccountID:   transaction.ToAccountID,
			Amount:        transaction.Amount,
			Status:        transaction.Status,
			CreatedAt:     transaction.CreatedAt,
			UpdatedAt:     transaction.UpdatedAt,
		}, nil
	}

	return transactionResponseDTO{
		ID:            updatedTransaction.ID,
		UserID:        updatedTransaction.UserID,
		FromAccountID: updatedTransaction.FromAccountID,
		ToAccountID:   updatedTransaction.ToAccountID,
		Amount:        updatedTransaction.Amount,
		Status:        updatedTransaction.Status,
		CreatedAt:     updatedTransaction.CreatedAt,
		UpdatedAt:     updatedTransaction.UpdatedAt,
	}, nil
}

// this function process the transaction, to be called after validating all values beforehand
func (uc *uc) processTransaction(ctx context.Context, transactionID, fromAccountID, toAccountID pgtype.UUID, amount decimal.Decimal) {
	// locking resource to prevent concurrent issues
	uc.mu.Lock()
	defer uc.mu.Unlock()

	// retrieving accounts to validate balances
	fromAccount, err := uc.repo.GetAccount(ctx, fromAccountID)
	if err != nil {
		slog.Error("error retrieving from account during processing", "error", err, "transaction_id", transactionID)
		uc.failTransaction(ctx, transactionID)
		return
	}
	toAccount, err := uc.repo.GetAccount(ctx, toAccountID)
	if err != nil {
		slog.Error("error retrieving to account during processing", "error", err, "transaction_id", transactionID)
		uc.failTransaction(ctx, transactionID)
		return
	}

	fromBalance := decimal.NewFromBigInt(fromAccount.Balance.Int, fromAccount.Balance.Exp)
	if fromBalance.LessThan(amount) {
		slog.Warn("insufficient balance during transaction processing", "transaction_id", transactionID, "from_account", fromAccountID)
		uc.failTransaction(ctx, transactionID)
		return
	}
	newFromBalance := fromBalance.Sub(amount)

	toBalance := decimal.NewFromBigInt(toAccount.Balance.Int, toAccount.Balance.Exp)
	newToBalance := toBalance.Add(amount)

	// updating accounts balances
	_, err = uc.repo.UpdateBalance(ctx, repo.UpdateBalanceParams{
		ID: fromAccountID,
		Balance: pgtype.Numeric{
			Int:   newFromBalance.Coefficient(),
			Exp:   newFromBalance.Exponent(),
			Valid: true,
		},
	})
	if err != nil {
		slog.Error("error updating from account balance", "error", err, "transaction_id", transactionID)
		uc.failTransaction(ctx, transactionID)
		return
	}

	_, err = uc.repo.UpdateBalance(ctx, repo.UpdateBalanceParams{
		ID: toAccountID,
		Balance: pgtype.Numeric{
			Int:   newToBalance.Coefficient(),
			Exp:   newToBalance.Exponent(),
			Valid: true,
		},
	})
	if err != nil {
		slog.Error("error updating to account balance", "error", err, "transaction_id", transactionID)
		// rollingback changes in case of error
		_, rollbackErr := uc.repo.UpdateBalance(ctx, repo.UpdateBalanceParams{
			ID:      fromAccountID,
			Balance: fromAccount.Balance,
		})
		if rollbackErr != nil {
			slog.Error("CRITICAL: failed to rollback from account balance", "error", rollbackErr, "transaction_id", transactionID)
		}
		uc.failTransaction(ctx, transactionID)
		return
	}

	// updating transaction status to compelted
	_, err = uc.repo.UpdateTransactionsStatus(ctx, repo.UpdateTransactionsStatusParams{
		ID:     transactionID,
		Status: repo.TransactionStatusCompleted,
	})
	if err != nil {
		slog.Error("error updating transaction status to completed", "error", err, "transaction_id", transactionID)
		return
	}

	slog.Info("transaction processed successfully", "transaction_id", transactionID)
}

// heloper function to return error at transaction
func (uc *uc) failTransaction(ctx context.Context, transactionID pgtype.UUID) {
	_, err := uc.repo.UpdateTransactionsStatus(ctx, repo.UpdateTransactionsStatusParams{
		ID:     transactionID,
		Status: repo.TransactionStatusFailed,
	})
	if err != nil {
		slog.Error("error updating transaction status to failed", "error", err, "transaction_id", transactionID)
	}
}

func (uc *uc) GetUserTransactions(ctx context.Context, userID pgtype.UUID, fromAccountID, toAccountID *pgtype.UUID, limit, offset int32) ([]transactionResponseDTO, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	transactions, err := uc.repo.GetUserTransactions(ctx, repo.GetUserTransactionsParams{
		UserID: userID,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		slog.Error("error fetching transactions for user", "error", err)
		return []transactionResponseDTO{}, nil
	}

	filteredTransactions := make([]transactionResponseDTO, 0, len(transactions))
	for _, transaction := range transactions {
		if fromAccountID != nil && transaction.FromAccountID.Bytes != fromAccountID.Bytes {
			continue
		}
		if toAccountID != nil && transaction.ToAccountID.Bytes != toAccountID.Bytes {
			continue
		}

		filteredTransactions = append(filteredTransactions, transactionResponseDTO{
			ID:            transaction.ID,
			UserID:        transaction.UserID,
			FromAccountID: transaction.FromAccountID,
			ToAccountID:   transaction.ToAccountID,
			Amount:        transaction.Amount,
			Status:        transaction.Status,
			CreatedAt:     transaction.CreatedAt,
			UpdatedAt:     transaction.UpdatedAt,
		})
	}

	return filteredTransactions, nil
}
