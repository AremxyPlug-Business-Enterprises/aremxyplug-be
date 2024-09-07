package handlers

import (
	"time"

	"github.com/aremxyplug-be/db"
	auth_pin "github.com/aremxyplug-be/lib/auth/pin"
	bankacc "github.com/aremxyplug-be/lib/bank/bank_acc"
	"github.com/aremxyplug-be/lib/bank/deposit"
	transactions "github.com/aremxyplug-be/lib/bank/transactions"
	"github.com/aremxyplug-be/lib/bank/transfer"
	elect "github.com/aremxyplug-be/lib/bills/electricity"
	"github.com/aremxyplug-be/lib/bills/tvsub"
	"github.com/aremxyplug-be/lib/emailclient"
	"github.com/aremxyplug-be/lib/key_generator"
	otpgen "github.com/aremxyplug-be/lib/otp_gen"
	pointredeem "github.com/aremxyplug-be/lib/point-redeem"
	"github.com/aremxyplug-be/lib/referral"
	"github.com/aremxyplug-be/lib/telcom/airtime"
	"github.com/aremxyplug-be/lib/telcom/data"
	"github.com/aremxyplug-be/lib/telcom/edu"

	"github.com/aremxyplug-be/config"
	"github.com/aremxyplug-be/lib/encryptor"
	"github.com/aremxyplug-be/lib/idgenerator"
	"github.com/aremxyplug-be/lib/timehelper"
	tokengenerator "github.com/aremxyplug-be/lib/tokekngenerator"
	uuidgenerator "github.com/aremxyplug-be/lib/uuidgeneraor"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
)

const (
	// email templates
	PasswordResetAlias = "password-reset"
	PasswordOTPAlias   = "password-otp"
	verifyEmailAlias   = "verify-email"
)

var validate = validator.New()

type HttpHandler struct {
	logger               *zap.Logger
	idGenerator          idgenerator.IdGenerator
	timeHelper           timehelper.TimeHelper
	store                db.DataStore
	secrets              *config.Secrets
	encrypt              encryptor.Encryptor
	jwt                  tokengenerator.TokenGenerator
	refreshTokenDuration time.Duration
	authTokenDuration    time.Duration
	uuidGenerator        uuidgenerator.UUIDGenerator
	emailClient          emailclient.EmailClient
	dataClient           *data.DataConn
	eduClient            *edu.EduConn
	vtuClient            *airtime.AirtimeConn
	tvClient             *tvsub.TvConn
	electClient          *elect.ElectricConn
	otp                  *otpgen.OTPConn
	virtualAcc           *bankacc.BankConfig
	bankTranc            *transactions.Transaction
	bankTrf              *transfer.Config
	bankDep              *deposit.Config
	referral             *referral.RefConfig
	point                *pointredeem.PointConfig
	pin                  *auth_pin.PinConfig
}

type HandlerOptions struct {
	Logger      *zap.Logger
	Store       db.DataStore
	Data        *data.DataConn
	Edu         *edu.EduConn
	VTU         *airtime.AirtimeConn
	TvSub       *tvsub.TvConn
	ElectSub    *elect.ElectricConn
	Secrets     *config.Secrets
	EmailClient emailclient.EmailClient
	Otp         *otpgen.OTPConn
	VirtualAcc  *bankacc.BankConfig
	BankTranc   *transactions.Transaction
	BankTrf     *transfer.Config
	BankDep     *deposit.Config
	Referral    *referral.RefConfig
	Point       *pointredeem.PointConfig
	Pin         *auth_pin.PinConfig
}

func NewHttpHandler(opt *HandlerOptions) *HttpHandler {
	refreshTokenDuration := calculateDefaultDuration(
		tokengenerator.RefreshTokenDuration,
		time.Duration(opt.Secrets.RefreshTokenDuration),
	)
	authTokenDuration := calculateDefaultDuration(
		tokengenerator.AuthTokenDuration,
		time.Duration(opt.Secrets.AuthTokenDuration),
	)

	tokenGeneratorPublicKey, err := key_generator.GeneratePublicKey(opt.Secrets.JWTPublicKey)
	if err != nil {
		opt.Logger.Error(
			"error parsing public key for token encryption",
			zap.Error(err),
		)
	}

	tokenGeneratorPrivateKey, err := key_generator.GeneratePrivateKey(opt.Secrets.JWTPrivateKey)
	if err != nil {
		opt.Logger.Error(
			"error parsing private key for token encryption",
			zap.Error(err),
		)
	}

	return &HttpHandler{
		logger:      opt.Logger,
		idGenerator: idgenerator.New(),
		timeHelper:  timehelper.New(),
		store:       opt.Store,
		secrets:     opt.Secrets,
		encrypt:     encryptor.NewEncryptor(),
		jwt: tokengenerator.New(
			tokenGeneratorPublicKey,
			tokenGeneratorPrivateKey,
		),
		refreshTokenDuration: refreshTokenDuration,
		authTokenDuration:    authTokenDuration,
		uuidGenerator:        uuidgenerator.NewGoogleUUIDGenerator(),
		eduClient:            opt.Edu,
		emailClient:          opt.EmailClient,
		dataClient:           opt.Data,
		vtuClient:            opt.VTU,
		tvClient:             opt.TvSub,
		electClient:          opt.ElectSub,
		otp:                  opt.Otp,
		virtualAcc:           opt.VirtualAcc,
		bankTranc:            opt.BankTranc,
		bankTrf:              opt.BankTrf,
		bankDep:              opt.BankDep,
		pin:                  opt.Pin,
		point:                opt.Point,
	}
}
