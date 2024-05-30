package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/sourcecd/gofermart/internal/auth"
	"github.com/sourcecd/gofermart/internal/config"
	"github.com/sourcecd/gofermart/internal/storage"
)

const (
	cookieMaxAge = 43200
)

func checkRequestCreds(r *http.Request) (*string, error) {
	if ck, err := r.Cookie("Bearer"); err == nil {
		return &ck.Value, nil
	}
	if bearer := r.Header.Get("Authorization"); bearer != "" {
		headerSlice := strings.Split(bearer, " ")
		if len(headerSlice) == 2 && headerSlice[0] == "Bearer" {
			bearer = headerSlice[1]
			return &bearer, nil
		}
	}
	return nil, errors.New("auth creds not found")
}

func register(ctx context.Context, seckey string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		testToken, err := auth.GenJWT(10, seckey)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.SetCookie(w, &http.Cookie{
			Name: "Bearer",
			Value: *testToken,
			MaxAge: cookieMaxAge,
		})

		token, err := checkRequestCreds(r)
		if err != nil {
			slog.Error(err.Error())
			http.Error(w, "401 Unauthorized", http.StatusUnauthorized)
			return
		}
		userid, err := auth.ExtractJWT(*token, seckey)
		if err != nil {
			slog.Error(err.Error())
			http.Error(w, "401 Unauthorized", http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprint("UserId: ", *userid, "\n")))
	}
}

func webRouter(ctx context.Context, seckey string) *chi.Mux {
	mux := chi.NewRouter()
	mux.Post("/api/user/register", register(ctx, seckey))

	return mux
}

func Run(ctx context.Context, config *config.Config) {
	db, err := storage.NewDB(config.Dsn)
	if err != nil {
		log.Fatal(err)
	}
	if err := db.PopulateDB(ctx); err != nil {
		log.Fatal(err)
	}
	if err := db.InitSecKey(ctx); err != nil {
		log.Fatal(err)
	}
	seckey, err := db.GetSecKey(ctx)
	if err != nil {
		log.Fatal(err)
	}

	http.ListenAndServe(":8080", webRouter(ctx, *seckey))
}
