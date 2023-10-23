package http

import (
	"net/http"

	"github.com/aremxyplug-be/lib/emailclient"
	"github.com/aremxyplug-be/lib/telcom/data"
	"github.com/aremxyplug-be/server/http/handlers"

	"github.com/aremxyplug-be/config"
	"github.com/aremxyplug-be/db"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/rs/cors"
	"go.uber.org/zap"
)

func MountServer(logger *zap.Logger, store db.DataStore, secrets *config.Secrets, emailClient emailclient.EmailClient, dataClient *data.DataConn) *chi.Mux {
	router := chi.NewRouter()

	// Middlewares
	router.Use(cors.New(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowCredentials: false,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		Debug:            true,
	}).Handler)
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			next.ServeHTTP(w, r)
		})
	})
	router.Use(middleware.Recoverer)
	router.Use(middleware.RequestID)
	router.Use(middleware.Logger)

	// Get handlers
	httpHandler := handlers.NewHttpHandler(&handlers.HandlerOptions{
		Logger:      logger,
		Store:       store,
		Secrets:     secrets,
		EmailClient: emailClient,
		Data:        dataClient,
	})

	// Routes
	// Health check
	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		render.Status(r, http.StatusOK)
		render.Data(w, r, []byte("Ok"))
	})

	router.Route("/api/v1", func(router chi.Router) {
		// SignUp
		router.Post("/signup", httpHandler.SignUp)
		// Login
		router.Post("/login", httpHandler.Login)
		// forgot password
		router.Post("/forgot-password", httpHandler.ForgotPassword)
		// reset password
		router.Patch("/reset-password", httpHandler.ResetPassword)

		router.Post("/send-otp", httpHandler.SendOTP)

		router.Post("/verify-otp", httpHandler.VerifyOTP)

		// test
		router.Post("/test", httpHandler.Testtoken)

		// buy data
		router.Post("/data", httpHandler.Data)
		// get user's transaction history
		router.Get("/data", httpHandler.Data)
		// get's  details of a transaction
		router.Get("/data/{id}", httpHandler.GetDataInfo)

		// returns all transactions: to be used by admins
		router.Get("/transactions", httpHandler.GetDataTransactions)

	})

	return router
}
