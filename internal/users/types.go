package users

import "github.com/jackc/pgx/v5/pgtype"

type createUserDTO struct {
	Cpf      string `json:"cpf" validate:"required,min=14,max=14"`
	Name     string `json:"name" validate:"required"`
	Password string `json:"password" validate:"required,min=8"`
}

type userResponse struct {
	Cpf  string `json:"cpf"`
	Name string `json:"name"`
}

type personalInformationResponse struct {
	ID   pgtype.UUID `json:"id"`
	Cpf  string      `json:"cpf"`
	Name string      `json:"name"`
}
