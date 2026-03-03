package main

import (
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/gutsavosouza/transactions-api/internal/accounts"
	repo "github.com/gutsavosouza/transactions-api/internal/adapters/postgres/sqlc"
	"github.com/gutsavosouza/transactions-api/internal/authentication"
	"github.com/gutsavosouza/transactions-api/internal/env"
	"github.com/gutsavosouza/transactions-api/internal/transactions"
	"github.com/gutsavosouza/transactions-api/internal/users"
	"github.com/gutsavosouza/transactions-api/internal/utils"
	"github.com/jackc/pgx/v5/pgxpool"
)

func (app *app) mount() http.Handler {
	var jwtSecretKey = env.GetString("JWT_SECRET", "somethingsecret")
	r := chi.NewRouter()

	r.Use(middleware.RequestID) // rate limiting
	r.Use(middleware.RealIP)    // rate limiting and analytics/tracing
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Use(middleware.Timeout(120 * time.Second))

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		utils.RespondWithJSON(w, 200, "OK")
	})

	r.Route("/v1", func(r chi.Router) {
		usersUseCase := users.NewUseCase(repo.New(app.db))
		usersHandlers := users.NewHandler(usersUseCase)

		authUseCase := authentication.NewUseCase(repo.New(app.db), jwtSecretKey)
		authHandlers := authentication.NewHandler(authUseCase)
		tokenMaker := authUseCase.GetTokenMaker()

		accountsUseCase := accounts.NewUseCase(repo.New(app.db))
		accountsHandlers := accounts.NewHandler(accountsUseCase)

		transactionsUseCase := transactions.NewUseCase(repo.New(app.db))
		transactionsHandlers := transactions.NewHandler(transactionsUseCase)

		r.Route("/users", func(r chi.Router) {
			r.Post("/", usersHandlers.NewUserHandler)
			r.With(authentication.GetAuthMiddleware(tokenMaker)).Get("/me", usersHandlers.MeHandler)
		})

		r.Route("/auth", func(r chi.Router) {
			r.Post("/login", authHandlers.LoginHandler)
		})

		r.Route("/account", func(r chi.Router) {
			r.Use(authentication.GetAuthMiddleware(tokenMaker))
			r.Post("/", accountsHandlers.NewAccount)
			r.Get("/", accountsHandlers.GetAllAccounts)
			r.Get("/{id}", accountsHandlers.GetAccount)
			r.Patch("/add-balance/{id}", accountsHandlers.AddBalance)
		})

		r.Route("/transaction", func(r chi.Router) {
			r.Use((authentication.GetAuthMiddleware(tokenMaker)))
			r.Post("/", transactionsHandlers.CreateTransaction)
			r.Get("/", transactionsHandlers.GetUserTransactions)
		})
	})

	return r
}

func (app *app) run(h http.Handler) error {
	server := &http.Server{
		Addr:         app.config.addr,
		Handler:      h,
		WriteTimeout: time.Second * 30,
		ReadTimeout:  time.Second * 10,
		IdleTimeout:  time.Minute,
	}

	log.Printf("server running on %s", app.config.addr)

	return server.ListenAndServe()
}

type app struct {
	config config
	// logger
	db *pgxpool.Pool
}

type config struct {
	addr string
	db   dbConfig
}

type dbConfig struct {
	dsn string // connection string
}
