package authentication

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/gutsavosouza/transactions-api/internal/token"
	"github.com/gutsavosouza/transactions-api/internal/utils"
)

type authKey struct{}

func GetAuthMiddleware(tokenMaker *token.JWTMaker) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// validate the token
			claims, err := verifyClaimsFromAuthHeader(r, tokenMaker)
			if err != nil {
				utils.RespondWithError(w, http.StatusUnauthorized, "locked resource")
				return
			}

			// pass the claims/payload down the context
			ctx := context.WithValue(r.Context(), authKey{}, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func verifyClaimsFromAuthHeader(r *http.Request, tokenMaker *token.JWTMaker) (*token.UserClaims, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return nil, fmt.Errorf("authorization header is missing")
	}

	fields := strings.Fields(authHeader)
	if len(fields) != 2 || fields[0] != "Bearer" {
		return nil, fmt.Errorf("invalid authorization header format")
	}

	token := fields[1]
	claims, err := tokenMaker.VerifyToken(token)
	if err != nil {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}

func GetClaimsFromContext(ctx context.Context) (*token.UserClaims, error) {
	claims, ok := ctx.Value(authKey{}).(*token.UserClaims)
	if !ok {
		return nil, fmt.Errorf("claims not found in context")
	}
	return claims, nil
}
