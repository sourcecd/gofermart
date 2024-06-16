package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/go-chi/chi/v5"
	"github.com/go-resty/resty/v2"
	"github.com/sourcecd/gofermart/internal/auth"
	"github.com/sourcecd/gofermart/internal/compression"
	"github.com/sourcecd/gofermart/internal/config"
	"github.com/sourcecd/gofermart/internal/logging"
	"github.com/sourcecd/gofermart/internal/models"
	"github.com/sourcecd/gofermart/internal/prjerrors"
	"github.com/sourcecd/gofermart/internal/retr"
	"github.com/sourcecd/gofermart/internal/storage"
	"golang.org/x/sync/errgroup"

	"github.com/theplant/luhn"
)

const (
	cookieMaxAge       = 43200
	pollInterval       = 1
	serverShutdownTime = 10
)

type handlers struct {
	ctx    context.Context
	seckey string
	db     storage.Store
	rtr    *retr.Retr
}

func checkRequestCreds(r *http.Request) (string, error) {
	if ck, err := r.Cookie("Bearer"); err == nil {
		return ck.Value, nil
	}
	if bearer := r.Header.Get("Authorization"); bearer != "" {
		headerSlice := strings.Split(bearer, " ")
		if len(headerSlice) == 2 && headerSlice[0] == "Bearer" {
			bearer = headerSlice[1]
			return bearer, nil
		}
	}
	return "", prjerrors.ErrAuthCredsNotFound
}

func userParse(r *http.Request) (*models.User, error) {
	regUser := &models.User{}
	enc := json.NewDecoder(r.Body)
	if err := enc.Decode(regUser); err != nil {
		return nil, prjerrors.ErrReqJSONParse
	}
	ok, err := govalidator.ValidateStruct(regUser)
	if err != nil {
		return nil, prjerrors.ErrValidateLogPass
	}
	if !ok {
		return nil, prjerrors.ErrValidateLogPass
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

		reg, err := userParse(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		id, err := h.rtr.UserFuncRetr(h.db.RegisterUser)(h.ctx, reg)
		if err != nil {
			if errors.Is(err, prjerrors.ErrAlreadyExists) {
				http.Error(w, err.Error(), http.StatusConflict)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		token, err := auth.GenerateJWT(id, h.seckey)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		SetTokenCookie(w, token)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(token))
	}
}

func (h *handlers) authUser() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := checkContentType(r, "application/json"); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		user, err := userParse(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		id, err := h.rtr.UserFuncRetr(h.db.AuthUser)(h.ctx, user)
		if err != nil {
			if errors.Is(err, prjerrors.ErrNotExists) {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		token, err := auth.GenerateJWT(id, h.seckey)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		SetTokenCookie(w, token)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(token))
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
		userid, err := auth.ExtractJWT(gettoken, h.seckey)
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

		if err := h.rtr.CreateOrderFuncRetr(h.db.CreateOrder)(h.ctx, userid, int64(ordnum)); err != nil {
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

func (h *handlers) ordersList() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		gettoken, err := checkRequestCreds(r)
		if err != nil {
			http.Error(w, "401 Unauthorized", http.StatusUnauthorized)
			return
		}
		userid, err := auth.ExtractJWT(gettoken, h.seckey)
		if err != nil {
			http.Error(w, "401 Unauthorized", http.StatusUnauthorized)
			return
		}
		var orderList []models.Order
		if err := h.rtr.ListOrdersFuncRetr(h.db.ListOrders)(h.ctx, userid, &orderList); err != nil {
			if errors.Is(err, prjerrors.ErrEmptyData) {
				http.Error(w, err.Error(), http.StatusNoContent)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := enc.Encode(orderList); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func (h *handlers) getBalance() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		gettoken, err := checkRequestCreds(r)
		if err != nil {
			http.Error(w, "401 Unauthorized", http.StatusUnauthorized)
			return
		}
		userid, err := auth.ExtractJWT(gettoken, h.seckey)
		if err != nil {
			http.Error(w, "401 Unauthorized", http.StatusUnauthorized)
			return
		}

		var balance models.Balance
		if err := h.rtr.GetBalanceFuncRetr(h.db.GetBalance)(h.ctx, userid, &balance); err != nil {
			return
		}

		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := enc.Encode(balance); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func (h *handlers) withdraw() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := checkContentType(r, "application/json"); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		gettoken, err := checkRequestCreds(r)
		if err != nil {
			http.Error(w, "401 Unauthorized", http.StatusUnauthorized)
			return
		}
		userid, err := auth.ExtractJWT(gettoken, h.seckey)
		if err != nil {
			http.Error(w, "401 Unauthorized", http.StatusUnauthorized)
			return
		}

		var withdraw models.Withdraw
		dec := json.NewDecoder(r.Body)
		if err := dec.Decode(&withdraw); err != nil {
			http.Error(w, err.Error(), http.StatusUnprocessableEntity)
			return
		}
		num, err := strconv.Atoi(withdraw.Order)
		if err != nil {
			http.Error(w, "number is not valid", http.StatusUnprocessableEntity)
			return
		}
		if !luhn.Valid(num) {
			http.Error(w, "luhn number is not valid", http.StatusUnprocessableEntity)
			return
		}
		if withdraw.Sum <= 0 {
			http.Error(w, "wrong withdraw sum", http.StatusUnprocessableEntity)
			return
		}

		if err := h.rtr.WithdrawFuncRetr(h.db.Withdraw)(h.ctx, userid, &withdraw); err != nil {
			if errors.Is(err, prjerrors.ErrNotEnough) {
				http.Error(w, err.Error(), http.StatusPaymentRequired)
				return
			}
			if errors.Is(err, prjerrors.ErrOrderAlreadyExists) {
				http.Error(w, err.Error(), http.StatusUnprocessableEntity)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("\n"))
	}
}

func (h *handlers) withdrawals() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		gettoken, err := checkRequestCreds(r)
		if err != nil {
			http.Error(w, "401 Unauthorized", http.StatusUnauthorized)
			return
		}
		userid, err := auth.ExtractJWT(gettoken, h.seckey)
		if err != nil {
			http.Error(w, "401 Unauthorized", http.StatusUnauthorized)
			return
		}

		var withdrawals []models.Withdrawals
		if err := h.rtr.WithdrawalsFuncRetr(h.db.Withdrawals)(h.ctx, userid, &withdrawals); err != nil {
			if errors.Is(err, prjerrors.ErrEmptyData) {
				http.Error(w, err.Error(), http.StatusNoContent)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := enc.Encode(withdrawals); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func webRouter(h *handlers) *chi.Mux {
	mux := chi.NewRouter()
	mux.Post("/api/user/register", logging.WriteLogging(compression.GzipCompressDecompress(h.registerUser())))
	mux.Post("/api/user/login", logging.WriteLogging(compression.GzipCompressDecompress(h.authUser())))
	mux.Post("/api/user/orders", logging.WriteLogging(compression.GzipCompressDecompress(h.orderRegister())))
	mux.Get("/api/user/orders", logging.WriteLogging(compression.GzipCompressDecompress(h.ordersList())))
	mux.Get("/api/user/balance", logging.WriteLogging(compression.GzipCompressDecompress(h.getBalance())))
	mux.Post("/api/user/balance/withdraw", logging.WriteLogging(compression.GzipCompressDecompress(h.withdraw())))
	mux.Get("/api/user/withdrawals", logging.WriteLogging(compression.GzipCompressDecompress(h.withdrawals())))

	return mux
}

func accuPoll(ctx context.Context, db *storage.PgDB, srv string) error {
	cl := resty.New().R()
	var orders []int64
	var listParsedOrders []models.Accrual

	if err := db.AccuPoll(ctx, &orders); err != nil {
		return err
	}

	for _, v := range orders {
		var parsedOrders models.Accrual
		resp, err := cl.Get(fmt.Sprintf("%s/api/orders/%d", srv, v))
		if err != nil {
			slog.Error(err.Error())
			continue
		}
		switch resp.StatusCode() {
		case http.StatusNoContent:
			continue
		case http.StatusTooManyRequests:
			hdr := resp.Header().Get("Retry-After")
			if hdr != "" {
				i, err := strconv.Atoi(hdr)
				if err != nil {
					time.Sleep(pollInterval * time.Second)
				}
				time.Sleep(time.Duration(i) * time.Second)
			}
		case http.StatusInternalServerError:
			slog.Error("unknown error: 500")
			continue
		}
		if err := json.Unmarshal(resp.Body(), &parsedOrders); err != nil {
			slog.Error(err.Error())
			continue
		}
		listParsedOrders = append(listParsedOrders, parsedOrders)
	}
	if err := db.AccuSave(ctx, listParsedOrders); err != nil {
		return err
	}
	return nil
}

func Run(ctx context.Context, config config.Config) {
	g, ctx := errgroup.WithContext(ctx)

	db, err := storage.NewDB(config.DatabaseDsn)
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

	rtr := retr.NewRetr()
	rtr.SetParams(1*time.Second, 30*time.Second, 3)

	h := &handlers{
		ctx:    ctx,
		seckey: seckey,
		db:     db,
		rtr:    rtr,
	}

	srv := http.Server{
		Addr:    config.ServerAddr,
		Handler: webRouter(h),
	}

	g.Go(func() error {
		logging.Slog.Info("Starting server on", slog.String("address", config.ServerAddr))
		return srv.ListenAndServe()
	})

	g.Go(func() error {
		<-ctx.Done()

		ctx, cancel := context.WithTimeout(context.Background(), serverShutdownTime*time.Second)
		defer cancel()

		return srv.Shutdown(ctx)
	})

	g.Go(func() error {
		for {
			select {
			case <-ctx.Done():
				return nil
			default:
			}
			if config.Accu != "" {
				if err := accuPoll(ctx, db, config.Accu); err != nil {
					slog.Error(err.Error())
				}
			} else {
				return errors.New("accrual system address empty")
			}
			time.Sleep(pollInterval * time.Second)
		}
	})

	if err := g.Wait(); err != nil {
		if errors.Is(err, http.ErrServerClosed) {
			logging.Slog.Info("Server successful shutdown")
			return
		}
		logging.Slog.Error(fmt.Sprintf("Failed: %s", err.Error()))
	}
}
