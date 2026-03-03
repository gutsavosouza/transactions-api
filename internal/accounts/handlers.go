package accounts

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
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

func (h *handler) NewAccount(w http.ResponseWriter, r *http.Request) {
	claims, err := authentication.GetClaimsFromContext(r.Context())
	if err != nil {
		slog.Error("error while retrieving claims from context: %v", "error", err)
		utils.RespondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	account, err := h.useCase.CreateAccount(r.Context(), claims.UserID)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "error creating account")
		return
	}

	utils.RespondWithJSON(w, http.StatusCreated, account)
}

func (h *handler) GetAccount(w http.ResponseWriter, r *http.Request) {
	claims, err := authentication.GetClaimsFromContext(r.Context())
	if err != nil {
		slog.Error("error while retrieving claims from context", "error", err)
		utils.RespondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	accountIDStr := chi.URLParam(r, "id")
	accountID, err := uuid.Parse(accountIDStr)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid id provided")
		return
	}

	account, err := h.useCase.GetAccount(r.Context(), pgtype.UUID{
		Bytes: accountID,
		Valid: true,
	})
	if err != nil {
		switch err {
		case ErrAccountNotFound:
			utils.RespondWithError(w, http.StatusNotFound, "account not found")
		default:
			utils.RespondWithError(w, http.StatusInternalServerError, "error fetching account")
		}
		return
	}

	if account.UserID.Bytes != claims.UserID.Bytes {
		utils.RespondWithError(w, http.StatusForbidden, "access denied")
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, account)
}

func (h *handler) GetAllAccounts(w http.ResponseWriter, r *http.Request) {
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

	accounts, err := h.useCase.GetAllAccounts(r.Context(), claims.UserID, limit, offset)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to fetch accounts")
	}

	utils.RespondWithJSON(w, http.StatusOK, accounts)
}

func (h *handler) AddBalance(w http.ResponseWriter, r *http.Request) {
	claims, err := authentication.GetClaimsFromContext(r.Context())
	if err != nil {
		slog.Error("error while retrieving claims from context", "error", err)
		utils.RespondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	accountIDStr := chi.URLParam(r, "id")
	accountID, err := uuid.Parse(accountIDStr)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid account id")
		return
	}

	amountDecimal := decimal.NewFromInt(1000)
	amount := pgtype.Numeric{
		Int:   amountDecimal.Coefficient(),
		Exp:   amountDecimal.Exponent(),
		Valid: true,
	}

	account, err := h.useCase.AddBalance(r.Context(), claims.UserID, pgtype.UUID{
		Bytes: accountID,
		Valid: true,
	}, amount)
	if err != nil {
		switch err {
		case ErrAccountNotFound:
			utils.RespondWithError(w, http.StatusNotFound, "account not found")
		case ErrUnauthorizedAccount:
			utils.RespondWithError(w, http.StatusForbidden, "you do not own this account")
		default:
			utils.RespondWithError(w, http.StatusInternalServerError, "error adding balance")
		}
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, account)
}
