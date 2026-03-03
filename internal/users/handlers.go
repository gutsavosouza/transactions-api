package users

import (
	"log/slog"
	"net/http"

	"github.com/gutsavosouza/transactions-api/internal/authentication"
	"github.com/gutsavosouza/transactions-api/internal/utils"
)

type handler struct {
	useCase UseCase
}

func NewHandler(useCase UseCase) *handler {
	return &handler{
		useCase: useCase,
	}
}

func (h *handler) NewUserHandler(w http.ResponseWriter, r *http.Request) {
	var newUserDTO createUserDTO
	if err := utils.ReadFromBody(r, &newUserDTO); err != nil {
		slog.Error("error while retrieving data from request body: %v", "error", err)
		utils.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := h.useCase.NewUser(r.Context(), newUserDTO)
	if err != nil {
		switch err {
		case ErrCPFAlreadyExists:
			utils.RespondWithError(w, http.StatusConflict, err.Error())
		case ErrInvalidCPF, ErrInvalidInput:
			utils.RespondWithError(w, http.StatusBadRequest, err.Error())
		default:
			utils.RespondWithError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	utils.RespondWithJSON(w, http.StatusCreated, user)
}

func (h *handler) MeHandler(w http.ResponseWriter, r *http.Request) {
	claims, err := authentication.GetClaimsFromContext(r.Context())
	if err != nil {
		slog.Error("error while retrieving claims from context: %v", "error", err)
		utils.RespondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, personalInformationResponse{
		ID:   claims.UserID,
		Cpf:  claims.Cpf,
		Name: claims.Name,
	})
}
