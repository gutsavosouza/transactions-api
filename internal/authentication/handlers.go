package authentication

import (
	"log/slog"
	"net/http"

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

func (h *handler) LoginHandler(w http.ResponseWriter, r *http.Request) {
	var loginDTO LoginDTO
	if err := utils.ReadFromBody(r, &loginDTO); err != nil {
		slog.Error("error while retrieving data from request body: %v", "error", err)
		utils.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	response, err := h.useCase.Login(r.Context(), loginDTO)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid credentials")
		return
	}

	utils.RespondWithJSON(w, 200, response)
}
