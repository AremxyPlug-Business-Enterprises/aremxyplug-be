package http

import (
	"net/http"

	"github.com/aremxyplug-be/lib/auth"
	auth_pin "github.com/aremxyplug-be/lib/auth/pin"
	bankacc "github.com/aremxyplug-be/lib/bank/bank_acc"
	"github.com/aremxyplug-be/lib/bank/deposit"
	"github.com/aremxyplug-be/lib/bank/transactions"
	"github.com/aremxyplug-be/lib/bank/transfer"
	elect "github.com/aremxyplug-be/lib/bills/electricity"
	"github.com/aremxyplug-be/lib/bills/tvsub"
	"github.com/aremxyplug-be/lib/emailclient"
	otpgen "github.com/aremxyplug-be/lib/otp_gen"
	pointredeem "github.com/aremxyplug-be/lib/point-redeem"
	"github.com/aremxyplug-be/lib/referral"
	"github.com/aremxyplug-be/lib/telcom/airtime"
	"github.com/aremxyplug-be/lib/telcom/data"
	"github.com/aremxyplug-be/lib/telcom/edu"
	"github.com/aremxyplug-be/server/http/handlers"

	"github.com/aremxyplug-be/config"
	"github.com/aremxyplug-be/db"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/rs/cors"
	"go.uber.org/zap"
)

type ServerConfig struct {
	Logger      *zap.Logger
	Store       db.DataStore
	Secrets     *config.Secrets
	EmailClient emailclient.EmailClient
	DataClient  *data.DataConn
	EduClient   *edu.EduConn
	Vtu         *airtime.AirtimeConn
	TvSub       *tvsub.TvConn
	ElectSub    *elect.ElectricConn
	Otp         *otpgen.OTPConn
	Auth        *auth.AuthConn
	VirtualAcc  *bankacc.BankConfig
	BankTranc   *transactions.Transaction
	BankTrf     *transfer.Config
	BankDep     *deposit.Config
	Referral    *referral.RefConfig
	Point       *pointredeem.PointConfig
	Pin         *auth_pin.PinConfig
}

func MountServer(config ServerConfig) *chi.Mux {
	router := chi.NewRouter()

	// Middlewares
	router.Use(cors.New(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowCredentials: false,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Authorization"},
		Debug:            true,
	}).Handler)
	router.Use(setJSONContentType)
	router.Use(middleware.Recoverer)
	router.Use(middleware.RequestID)
	router.Use(middleware.Logger)

	// Get handlers
	httpHandler := handlers.NewHttpHandler(&handlers.HandlerOptions{
		Logger:      config.Logger,
		Store:       config.Store,
		Secrets:     config.Secrets,
		EmailClient: config.EmailClient,
		Data:        config.DataClient,
		Edu:         config.EduClient,
		VTU:         config.Vtu,
		TvSub:       config.TvSub,
		ElectSub:    config.ElectSub,
		Otp:         config.Otp,
		VirtualAcc:  config.VirtualAcc,
		BankTranc:   config.BankTranc,
		BankTrf:     config.BankTrf,
		BankDep:     config.BankDep,
		Referral:    config.Referral,
		Point:       config.Point,
		Pin:         config.Pin,
	})

	// Routes
	// Health check
	router.Get("/health", healthCheck)

	router.Route("/api/v1", func(router chi.Router) {
		// SignUp
		router.Post("/signup", httpHandler.SignUp)
		// Login
		router.Post("/login", httpHandler.Login)
		// forgot password
		router.Post("/forgot-password", httpHandler.ForgotPassword)
		// reset password
		router.Patch("/reset-password", httpHandler.ResetPassword)

		router.Get("/verify-token", httpHandler.ValidateToken)

		router.Post("/send-otp", httpHandler.SendOTP)

		router.Post("/verify-otp", httpHandler.VerifyOTP)

		// test
		router.Post("/test", httpHandler.Testtoken)

		router.Get("/banks", httpHandler.GetBanks)

		router.Get("/deposit", httpHandler.DepositAccount)

		authRouter := router.With(config.Auth.Authorize)
		// Data Routes
		dataRoutes(router, httpHandler)
		// smile data routes
		smileDataRoutes(authRouter, httpHandler)
		// spectranet data routes
		spectranetDataRoutes(authRouter, httpHandler)

		// Edu Routes
		eduRoutes(router, httpHandler)

		//  Airtime Routes
		airtimeRoutes(router, httpHandler)

		// TvSubscription Routes
		tvSubscriptionRoutes(router, httpHandler)

		// Electricity bills routes
		electricityBillRoutes(router, httpHandler)

		// bank routes
		bankRoutes(authRouter, httpHandler)

		pinRoute(authRouter, httpHandler)

		extraRoutes(router, httpHandler)

		virtualAccRoutes(authRouter, httpHandler)
		/*
			transferMoneyRoutes(authRouter, httpHandler)

			depositRoutes(authRouter, httpHandler)
		*/
	})

	return router
}

func setJSONContentType(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	render.Status(r, http.StatusOK)
	render.Data(w, r, []byte("Ok"))
}

func dataRoutes(r chi.Router, httpHandler *handlers.HttpHandler) {
	r.Route("/data", func(router chi.Router) {
		router.Post("/", httpHandler.Data)
		router.Get("/", httpHandler.Data)
		router.Get("/{id}", httpHandler.GetDataInfo)
		router.Get("/transactions", httpHandler.GetDataTransactions)

		router.Route("/recipient", func(route chi.Router) {
			route.Post("/", httpHandler.TelcomRecipient)
			route.Get("/", httpHandler.TelcomRecipient)
			route.Put("/", httpHandler.TelcomRecipient)
			route.Delete("/", httpHandler.TelcomRecipient)
		})
	})
}

func smileDataRoutes(r chi.Router, httpHandler *handlers.HttpHandler) {
	r.Route("/data/smile", func(router chi.Router) {
		router.Post("/", httpHandler.SmileData)
		router.Get("/", httpHandler.SmileData)
		router.Get("/{id}", httpHandler.GetSmileDataDetails)
		router.Get("/transactions", httpHandler.GetSmileTransactions)
	})
}

func spectranetDataRoutes(r chi.Router, httpHandler *handlers.HttpHandler) {
	r.Route("/data/spectranet", func(router chi.Router) {
		router.Post("/", httpHandler.SpectranetData)
		router.Get("/", httpHandler.SpectranetData)
		router.Get("/{id}", httpHandler.GetSpecDataDetails)
		router.Get("/transactions", httpHandler.GetSpectranetTransactions)
	})
}

func eduRoutes(r chi.Router, httpHandler *handlers.HttpHandler) {
	r.Route("/edu", func(router chi.Router) {
		router.Post("/", httpHandler.EduPins)
		router.Get("/", httpHandler.EduPins)
		router.Get("/{id}", httpHandler.GetDataInfo)
		router.Get("/transactions", httpHandler.GetEduTransactions)
	})
}

func airtimeRoutes(r chi.Router, httpHandler *handlers.HttpHandler) {
	r.Route("/airtime", func(router chi.Router) {
		router.Post("/", httpHandler.Airtime)
		router.Get("/", httpHandler.Airtime)
		router.Get("/{id}", httpHandler.GetAirtimeInfo)
		router.Get("/transactions", httpHandler.GetAirtimeTransactions)

		router.Route("/recipient", func(route chi.Router) {
			route.Post("/", httpHandler.TelcomRecipient)
			route.Get("/", httpHandler.TelcomRecipient)
			route.Put("/", httpHandler.TelcomRecipient)
			route.Delete("/", httpHandler.TelcomRecipient)
		})
	})
}

func tvSubscriptionRoutes(r chi.Router, httpHandler *handlers.HttpHandler) {
	r.Route("/tvsub", func(router chi.Router) {
		router.Post("/", httpHandler.TVSubscriptions)
		router.Get("/", httpHandler.TVSubscriptions)
		router.Get("/{id}", httpHandler.GetTvSubDetails)
		router.Get("/transactions", httpHandler.GetTvSubscriptions)
	})
}

func electricityBillRoutes(r chi.Router, httpHandler *handlers.HttpHandler) {
	r.Route("/electric-bill", func(router chi.Router) {
		router.Post("/", httpHandler.ElectricBill)
		router.Get("/", httpHandler.ElectricBill)
		router.Get("/{id}", httpHandler.GetElectricBillDetails)
		router.Get("/transactions", httpHandler.GetElectricBills)
	})
}

func bankRoutes(r chi.Router, httpHandler *handlers.HttpHandler) {
	r.Route("/bank", func(router chi.Router) {
		router.Route("/transfer", func(router chi.Router) {
			router.Post("/", httpHandler.Transfer)
			router.Get("/", httpHandler.Transfer)
			router.Get("/{id}", httpHandler.GetTransferDetails)
		})
		router.Route("/deposit", func(router chi.Router) {
			router.Get("/", httpHandler.GetDepositHistory)
			router.Get("/{id}", httpHandler.GetDepositDetail)
		})
		router.Get("/transactions", httpHandler.GetAllBankTransactions)
	})
}

func pinRoute(r chi.Router, httpHandler *handlers.HttpHandler) {
	r.Route("/pin", func(router chi.Router) {
		router.Post("/", httpHandler.Pin)
		router.Patch("/", httpHandler.Pin)
		router.Post("/verify", httpHandler.VerifyPIN)
	})
}

func extraRoutes(r chi.Router, httpHandler *handlers.HttpHandler) {
	r.Route("/extra", func(router chi.Router) {
		router.Route("/referral", func(router chi.Router) {
			router.Get("/", httpHandler.Referral)
			router.Post("/", httpHandler.Referral)
		})
		router.Route("/point", func(router chi.Router) {
			router.Get("/", httpHandler.Points)
			router.Post("/", httpHandler.Points)
		})
	})
}

func virtualAccRoutes(r chi.Router, httpHandler *handlers.HttpHandler) {
	r.Route("/virtualacc", func(router chi.Router) {
		r.Get("/", httpHandler.VirtualAccount)
		r.Post("/", httpHandler.VirtualAccount)
	})
}

/*
func depositRoutes(r chi.Router, httpHandler *handlers.HttpHandler) {
	r.Route("/deposit", func(router chi.Router) {
		router.Get("/", httpHandler.GetDepositHistory)
		router.Get("/{id}", httpHandler.GetDepositDetail)
	})
}
*/
