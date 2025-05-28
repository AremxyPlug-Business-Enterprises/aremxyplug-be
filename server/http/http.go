package http

import (
	"net/http"

	"github.com/aremxyplug-be/db/sqlstore"
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
	"github.com/aremxyplug-be/lib/smsclient/termii"
	"github.com/aremxyplug-be/lib/telcom/airtime"
	"github.com/aremxyplug-be/lib/telcom/data"
	"github.com/aremxyplug-be/lib/telcom/edu"
	"github.com/aremxyplug-be/lib/verification"
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
	Logger       *zap.Logger
	Store        db.DataStore
	SqlStore     *sqlstore.SqlStore
	Secrets      *config.Secrets
	EmailClient  emailclient.EmailClient
	DataClient   *data.DataConn
	EduClient    *edu.EduConn
	Vtu          *airtime.AirtimeConn
	TvSub        *tvsub.TvConn
	ElectSub     *elect.ElectricConn
	Otp          *otpgen.OTPConn
	Auth         *auth.AuthConn
	VirtualAcc   *bankacc.BankConfig
	BankTranc    *transactions.Transaction
	BankTrf      *transfer.Config
	BankDep      *deposit.Config
	Point        *pointredeem.PointConfig
	Pin          *auth_pin.PinConfig
	SmsClient    *termii.SMSConn
	VerifyClient verification.VerificationClient
}

func MountServer(config ServerConfig) *chi.Mux {
	router := chi.NewRouter()

	allowedOrigin := []string{"http://localhost:3000", "https://test.aremxyplug.com", "https://aremxyplug.com"}

	// Middlewares
	router.Use(cors.New(cors.Options{
		AllowedOrigins:   allowedOrigin,
		AllowCredentials: true,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "Origin"},
		ExposedHeaders:   []string{"Authorization"},
		Debug:            true,
	}).Handler)
	router.Use(setJSONContentType)
	router.Use(middleware.Recoverer)
	router.Use(middleware.RequestID)
	router.Use(middleware.Logger)

	// Get handlers
	httpHandler := handlers.NewHttpHandler(&handlers.HandlerOptions{
		Logger:       config.Logger,
		Store:        config.Store,
		SqlStore:     config.SqlStore,
		Secrets:      config.Secrets,
		EmailClient:  config.EmailClient,
		Data:         config.DataClient,
		Edu:          config.EduClient,
		VTU:          config.Vtu,
		TvSub:        config.TvSub,
		ElectSub:     config.ElectSub,
		Otp:          config.Otp,
		VirtualAcc:   config.VirtualAcc,
		BankTranc:    config.BankTranc,
		BankTrf:      config.BankTrf,
		BankDep:      config.BankDep,
		Point:        config.Point,
		Pin:          config.Pin,
		SMSClient:    config.SmsClient,
		VerifyClient: config.VerifyClient,
	})

	// Routes
	// Health check
	router.Get("/health", healthCheck)
	router.Post("/webhook", handlers.WebhookHandler)

	router.Route("/api/v1", func(router chi.Router) {
		// SignUp
		router.Post("/signup", httpHandler.SignUp)
		// Login
		router.Post("/login", httpHandler.Login)
		// forgot password
		router.Post("/forgot-password", httpHandler.ForgotPassword)

		router.Get("/verify-token", httpHandler.ValidateToken)

		sendOTPRoutes(router, httpHandler)

		smsRoutes(router, httpHandler)

		verifyOTPRoutes(router, httpHandler)

		authRouter := router.With(config.Auth.Authorize)
		// reset password
		authRouter.Patch("/reset-password", httpHandler.ResetPassword)

		authRouter.Patch("/update-password", httpHandler.UpdatePassword)
		// Data Routes
		dataRoutes(authRouter, httpHandler)
		// smile data routes
		smileDataRoutes(authRouter, httpHandler)
		// spectranet data routes
		spectranetDataRoutes(authRouter, httpHandler)

		authRouter.Post("/verify", httpHandler.VerifyIdentity)

		// Edu Routes
		eduRoutes(authRouter, httpHandler)

		//  Airtime Routes
		airtimeRoutes(authRouter, httpHandler)

		// TvSubscription, Electricity bills Routes
		billRoutes(authRouter, httpHandler)

		// bank routes
		bankRoutes(authRouter, httpHandler)

		pinRoute(authRouter, httpHandler)

		extraRoutes(authRouter, httpHandler)

		virtualAccRoutes(authRouter, httpHandler)

		getBalance(authRouter, httpHandler)

		checkVerification(authRouter, httpHandler)

		productRoutes(authRouter, httpHandler)

		authRouter.Get("/chart", httpHandler.Chart)

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

func billRoutes(r chi.Router, httpHandler *handlers.HttpHandler) {
	r.Route("/bills", func(router chi.Router) {
		router.Route("/tvsub", func(ro chi.Router) {
			ro.Post("/", httpHandler.TVSubscriptions)
			ro.Get("/", httpHandler.TVSubscriptions)
			ro.Get("/{id}", httpHandler.GetTvSubDetails)
			ro.Get("/transactions", httpHandler.GetTvSubscriptions)
		})

		router.Route("/electric-bill", func(ro chi.Router) {
			ro.Post("/", httpHandler.ElectricBill)
			ro.Get("/", httpHandler.ElectricBill)
			ro.Get("/{id}", httpHandler.GetElectricBillDetails)
			ro.Get("/transactions", httpHandler.GetElectricBills)
		})
		router.Post("/verify", httpHandler.VerifyBill)
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
		router.Put("/reset", httpHandler.ResetPin)
	})

}

func extraRoutes(r chi.Router, httpHandler *handlers.HttpHandler) {
	r.Route("/extra", func(router chi.Router) {
		router.Route("/referral", func(router chi.Router) {
			router.Get("/", httpHandler.ReferralCode)
			router.Route("/referred-users", func(ro chi.Router) {
				ro.Get("/", httpHandler.Referral)
			})
		})
		router.Route("/point", func(router chi.Router) {
			router.Get("/", httpHandler.Points)
			router.Post("/", httpHandler.Points)
		})
	})
}

func virtualAccRoutes(r chi.Router, httpHandler *handlers.HttpHandler) {
	r.Route("/virtualacc", func(router chi.Router) {
		router.Get("/", httpHandler.VirtualAccount)
		router.Post("/", httpHandler.VirtualAccount)
	})
}

func verifyOTPRoutes(r chi.Router, httpHandler *handlers.HttpHandler) {
	r.Route("/verify-otp", func(router chi.Router) {
		router.Post("/signin", httpHandler.VerifyOTP)
		router.Post("/signup", httpHandler.VerifyOTP)
		router.Post("/resetpassword", httpHandler.VerifyOTP)
		router.Post("/resetpin", httpHandler.VerifyOTP)
	})
}

func sendOTPRoutes(r chi.Router, httpHandler *handlers.HttpHandler) {
	r.Route("/send-otp", func(router chi.Router) {
		router.Post("/signin", httpHandler.SendOTP)
		router.Post("/signup", httpHandler.SendOTP)
		router.Post("/resetpassword", httpHandler.SendOTP)
		router.Post("/resetpin", httpHandler.SendOTP)
	})
}

func smsRoutes(r chi.Router, httpHandler *handlers.HttpHandler) {
	r.Route("/sms", func(router chi.Router) {
		router.Route("/send", func(router chi.Router) {
			router.Post("/", httpHandler.SendSMSOTP)
		})
		router.Route("/verify", func(router chi.Router) {
			router.Post("/signup", httpHandler.VerifySMSOTP)
			router.Post("/signin", httpHandler.VerifySMSOTP)
			router.Post("/resetpassword", httpHandler.VerifySMSOTP)
		})
	})
}

func getBalance(r chi.Router, httpHandler *handlers.HttpHandler) {
	r.Route("/balance", func(router chi.Router) {
		router.Get("/", httpHandler.GetBalance)
	})
}

func checkVerification(router chi.Router, httpHandler *handlers.HttpHandler) {
	router.Route("/check-verification", func(r chi.Router) {
		r.Get("/", httpHandler.CheckVerification)
	})
}

func productRoutes(r chi.Router, httpHandler *handlers.HttpHandler) {
	r.Route("/products", func(router chi.Router) {
		// TV Subscription Routes
		router.Route("/tvsub/{product}", func(router chi.Router) {
			router.Post("/", httpHandler.TvSubHandler)       // POST /tvsub/dstv
			router.Get("/", httpHandler.TvSubHandler)        // GET /tvsub/dstv (all subscriptions)
			router.Patch("/{id}", httpHandler.TvSubHandler)  // PATCH /tvsub/dstv/2
			router.Delete("/{id}", httpHandler.TvSubHandler) // DELETE /tvsub/dstv/2
		})

		// Telecom Plans Routes
		router.Route("/telecom", func(router chi.Router) {
			router.Get("/list/{networkID}", httpHandler.TelcomProducts)
			router.Post("/", httpHandler.TelecomPlans)
			router.Get("/{productID}", httpHandler.TelecomPlans)
			router.Patch("/{planID}", httpHandler.TelecomPlans)
			router.Delete("/{planID}", httpHandler.TelecomPlans)
		})

		// Electric Subscription Routes

		// Education Pin Routes
	})
}
