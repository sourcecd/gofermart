package server

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/asaskevich/govalidator"
	"github.com/go-chi/chi/v5"
	"github.com/sourcecd/gofermart/internal/auth"
	"github.com/sourcecd/gofermart/internal/config"
	"github.com/sourcecd/gofermart/internal/models"
	"github.com/sourcecd/gofermart/internal/prjerrors"
	"github.com/sourcecd/gofermart/internal/storage"
)

const (
	cookieMaxAge = 43200
)

type handlers struct {
	ctx    context.Context
	seckey string
	db     *storage.PgDB
}

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

func UserParse(r *http.Request) (*models.User, error) {
	regUser := &models.User{}
	enc := json.NewDecoder(r.Body)
	if err := enc.Decode(regUser); err != nil {
		return nil, errors.New("request json parse failed")
	}
	ok, err := govalidator.ValidateStruct(regUser)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.New("validate login or password false (maybe empty)")
	}
	return regUser, nil
}

func SetTokenCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:   "Bearer",
		Value:  token,
		MaxAge: cookieMaxAge,
	})
}

func checkContentType(r *http.Request) error {
	if r.Header.Get("Content-Type") != "application/json" {
		return errors.New("wrong content type")
	}
	return nil
}

func (h *handlers) registerUser() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := checkContentType(r); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		reg, err := UserParse(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		id, err := h.db.RegisterUser(h.ctx, reg)
		if err != nil {
			if errors.Is(err, prjerrors.ErrAlreadyExists) {
				http.Error(w, err.Error(), http.StatusConflict)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		token, err := auth.GenJWT(*id, h.seckey)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		SetTokenCookie(w, *token)

		//MOVE_TO_LOGIN
		/*gettoken, err := checkRequestCreds(r)
		if err != nil {
			slog.Error(err.Error())
			http.Error(w, "401 Unauthorized", http.StatusUnauthorized)
			return
		}*/

		/*userid, err := auth.ExtractJWT(*gettoken, seckey)
		if err != nil {
			slog.Error(err.Error())
			http.Error(w, "401 Unauthorized", http.StatusUnauthorized)
			return
		}*/

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(*token))
	}
}

func (h *handlers) authUser() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := checkContentType(r); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		user, err := UserParse(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		id, err := h.db.AuthUser(h.ctx, user)
		if err != nil {
			if errors.Is(err, prjerrors.ErrNotExists) {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		token, err := auth.GenJWT(*id, h.seckey)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		SetTokenCookie(w, *token)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(*token))
	}
}

func webRouter(h *handlers) *chi.Mux {
	mux := chi.NewRouter()
	mux.Post("/api/user/register", h.registerUser())
	mux.Post("/api/user/login", h.authUser())

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

	h := &handlers{
		ctx:    ctx,
		seckey: *seckey,
		db:     db,
	}

	http.ListenAndServe(":8080", webRouter(h))
}
