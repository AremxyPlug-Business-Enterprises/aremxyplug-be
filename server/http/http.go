package http

import (
	"github.com/aremxyplug-be/lib/emailclient"
	"github.com/aremxyplug-be/server/http/handlers"
	"net/http"

	"github.com/aremxyplug-be/config"
	"github.com/aremxyplug-be/db"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/rs/cors"
	"go.uber.org/zap"
)

func MountServer(logger *zap.Logger, store db.DataStore, secrets *config.Secrets, emailClient emailclient.EmailClient) *chi.Mux {
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
		// password reset
		router.Post("/password-reset", httpHandler.PasswordReset)

		router.Post("/send-otp", httpHandler.SendOTP)

		router.Post("/verify-otp", httpHandler.VerifyOTP)

		// test
		router.Post("/test", httpHandler.Testtoken)

	})

	return router
}
