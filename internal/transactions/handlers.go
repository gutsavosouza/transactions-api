package transactions

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/gutsavosouza/transactions-api/internal/authentication"
	"github.com/gutsavosouza/transactions-api/internal/utils"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"
)

type handler struct {
	useCase UseCase
}

func NewHandler(useCase UseCase) *handler {
	return &handler{
		useCase: useCase,
	}
}

func (h *handler) CreateTransaction(w http.ResponseWriter, r *http.Request) {
	claims, err := authentication.GetClaimsFromContext(r.Context())
	if err != nil {
		slog.Error("error while retrieving claims from context", "error", err)
		utils.RespondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var newTransactionDTO createTransactionDTO
	if err := utils.ReadFromBody(r, &newTransactionDTO); err != nil {
		slog.Error("error while retrieving data from request body: %v", "error", err)
		utils.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	fromAccountUUID, err := uuid.Parse(newTransactionDTO.FromAccountID)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid from_account_id")
		return
	}
	toAccountUUID, err := uuid.Parse(newTransactionDTO.ToAccountID)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid to_account_id")
		return
	}
	if newTransactionDTO.Amount <= 0 {
		utils.RespondWithError(w, http.StatusBadRequest, "amount must be greater than zero")
		return
	}
	amountDecimal := decimal.NewFromFloat(newTransactionDTO.Amount).Round(2)
	// slog.Debug("amount decimal", "value", amountDecimal)
	amount := pgtype.Numeric{
		Int:   amountDecimal.Coefficient(),
		Exp:   amountDecimal.Exponent(),
		Valid: true,
	}
	// slog.Debug("amount", "value", amount)

	transaction, err := h.useCase.CreateTransaction(
		r.Context(),
		claims.UserID,
		pgtype.UUID{Bytes: fromAccountUUID, Valid: true},
		pgtype.UUID{Bytes: toAccountUUID, Valid: true},
		amount,
	)
	if err != nil {
		switch err {
		case ErrSameAccount:
			utils.RespondWithError(w, http.StatusBadRequest, "from and to accounts cannot be the same")
		case ErrInvalidAmount:
			utils.RespondWithError(w, http.StatusBadRequest, "invalid amount")
		case ErrAccountNotFound:
			utils.RespondWithError(w, http.StatusNotFound, "account not found")
		case ErrInsufficientBalance:
			utils.RespondWithError(w, http.StatusBadRequest, "insufficient balance")
		case ErrUnauthorizedAccount:
			utils.RespondWithError(w, http.StatusForbidden, "you do not own the from account")
		default:
			utils.RespondWithError(w, http.StatusInternalServerError, "error creating transaction")
		}
		return
	}

	utils.RespondWithJSON(w, http.StatusCreated, transaction)
}

func (h *handler) GetUserTransactions(w http.ResponseWriter, r *http.Request) {
	claims, err := authentication.GetClaimsFromContext(r.Context())
	if err != nil {
		slog.Error("error while retrieving claims from context", "error", err)
		utils.RespondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	limit := int32(10)
	offset := int32(0)
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		parsedLimit, err := strconv.ParseInt(limitStr, 10, 32)
		if err != nil || parsedLimit < 1 {
			utils.RespondWithError(w, http.StatusBadRequest, "invalid limit parameter")
			return
		}
		limit = int32(parsedLimit)
	}
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		parsedOffset, err := strconv.ParseInt(offsetStr, 10, 32)
		if err != nil || parsedOffset < 0 {
			utils.RespondWithError(w, http.StatusBadRequest, "invalid offset parameter")
			return
		}
		offset = int32(parsedOffset)
	}

	var fromAccountID *pgtype.UUID
	var toAccountID *pgtype.UUID
	if fromAccountStr := r.URL.Query().Get("from_account_id"); fromAccountStr != "" {
		fromUUID, err := uuid.Parse(fromAccountStr)
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, "invalid from_account_id parameter")
			return
		}
		fromAccountID = &pgtype.UUID{Bytes: fromUUID, Valid: true}
	}
	if toAccountStr := r.URL.Query().Get("to_account_id"); toAccountStr != "" {
		toUUID, err := uuid.Parse(toAccountStr)
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, "invalid to_account_id parameter")
			return
		}
		toAccountID = &pgtype.UUID{Bytes: toUUID, Valid: true}
	}

	transactions, err := h.useCase.GetUserTransactions(r.Context(), claims.UserID, fromAccountID, toAccountID, limit, offset)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to fetch transactions")
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, transactions)
}
