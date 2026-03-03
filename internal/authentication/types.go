package authentication

type LoginDTO struct {
	Cpf      string `json:"cpf"`
	Password string `json:"password"`
}

type LoginResponseDTO struct {
	AccessToken string `json:"access_token"`
}
