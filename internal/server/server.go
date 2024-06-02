package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/asaskevich/govalidator"
	"github.com/go-chi/chi/v5"
	"github.com/sourcecd/gofermart/internal/auth"
	"github.com/sourcecd/gofermart/internal/config"
	"github.com/sourcecd/gofermart/internal/models"
	"github.com/sourcecd/gofermart/internal/prjerrors"
	"github.com/sourcecd/gofermart/internal/storage"

	"github.com/theplant/luhn"
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

func checkContentType(r *http.Request, contentType string) error {
	if r.Header.Get("Content-Type") != contentType {
		return errors.New("wrong content type")
	}
	return nil
}

func (h *handlers) registerUser() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := checkContentType(r, "application/json"); err != nil {
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

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(*token))
	}
}

func (h *handlers) authUser() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := checkContentType(r, "application/json"); err != nil {
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

func (h *handlers) orderRegister() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := checkContentType(r, "text/plain"); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		gettoken, err := checkRequestCreds(r)
		if err != nil {
			http.Error(w, "401 Unauthorized", http.StatusUnauthorized)
			return
		}
		userid, err := auth.ExtractJWT(*gettoken, h.seckey)
		if err != nil {
			http.Error(w, "401 Unauthorized", http.StatusUnauthorized)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		ordnum, err := strconv.Atoi(string(body))
		if err != nil {
			http.Error(w, "order number is not number", http.StatusBadRequest)
			return
		}
		if !luhn.Valid(ordnum) {
			http.Error(w, "luhn number is not valid", http.StatusUnprocessableEntity)
			return
		}

		if err := h.db.CreateOrder(h.ctx, *userid, ordnum); err != nil {
			if errors.Is(err, prjerrors.ErrOrderAlreadyExists) {
				http.Error(w, err.Error(), http.StatusOK)
				return
			}
			if errors.Is(err, prjerrors.ErrOtherOrderAlreadyExists) {
				http.Error(w, err.Error(), http.StatusConflict)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(fmt.Sprint(ordnum)))
	}
}

func webRouter(h *handlers) *chi.Mux {
	mux := chi.NewRouter()
	mux.Post("/api/user/register", h.registerUser())
	mux.Post("/api/user/login", h.authUser())
	mux.Post("/api/user/orders", h.orderRegister())

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
