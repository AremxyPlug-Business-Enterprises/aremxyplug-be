package otpgen

import (
	"context"
	"math/rand"
	"time"

	"github.com/aremxyplug-be/db"
	"github.com/aremxyplug-be/db/models"
	"github.com/pquerna/otp/totp"
	"go.uber.org/zap"
)

type OTPConn struct {
	dbconn db.Extras
	logger *zap.Logger
}

func NewOTP(store db.DataStore, logger *zap.Logger) *OTPConn {
	return &OTPConn{
		dbconn: store,
		logger: logger,
	}
}

func (o *OTPConn) GenerateOTP(ctx context.Context, channel, target string) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	o.logger.Info("Generating OTP", zap.String("channel", channel), zap.String("target", target))

	key, err := totp.Generate(
		totp.GenerateOpts{
			Issuer:      "AremxyPlug",
			AccountName: target,
			Period:      300,
			Digits:      6,
		},
	)
	if err != nil {
		o.logger.Error("Failed to generate OTP key", zap.Error(err))
		return "", err
	}

	now := time.Now()

	data := models.OTP{
		Secret:  key.Secret(),
		Channel: channel,
		Target:  target,
	}

	if err := o.dbconn.SaveOTP(ctx, data); err != nil {
		o.logger.Error("Failed to save OTP", zap.Error(err))
		return "", err
	}

	otp, err := totp.GenerateCodeCustom(key.Secret(), now, totp.ValidateOpts{
		Period: 300,
		Digits: 6,
	})
	if err != nil {
		o.logger.Error("Failed to generate OTP code", zap.Error(err))
		return "", err
	}

	o.logger.Info("OTP generated successfully", zap.String("otp", otp))
	return otp, nil
}

func (o *OTPConn) ValidateOTP(ctx context.Context, otp, channel, target string) (bool, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	o.logger.Info("Validating OTP", zap.String("channel", channel), zap.String("target", target), zap.String("otp", otp))

	data, err := o.dbconn.GetOTP(ctx, channel, target)
	if err != nil {
		o.logger.Error("Failed to get OTP from database", zap.Error(err))
		return false, err
	}

	now := time.Now()

	valid, err := totp.ValidateCustom(otp, data.Secret, now, totp.ValidateOpts{
		Period: 300,
		Digits: 6,
	})
	if err != nil {
		o.logger.Error("Failed to validate OTP", zap.Error(err))
		return false, err
	}
	if !valid {
		o.logger.Warn("Invalid OTP", zap.String("otp", otp))
		return false, nil
	}

	o.logger.Info("OTP validated successfully")
	return true, nil
}

// GenerateID generates a unique 8-digit ID
func (o *OTPConn) GenerateID(ctx context.Context) (int, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	o.logger.Info("Generating unique ID")

	var id int
	var unique bool
	var err error

	for {
		id = rand.Intn(90000000) + 10000000 // Generates a number between 10000000 and 99999999
		unique, err = o.isIDUnique(ctx, id)
		if err != nil {
			o.logger.Error("Failed to check ID uniqueness", zap.Error(err))
			return 0, err
		}
		if unique {
			break
		}
	}

	o.logger.Info("Unique ID generated successfully", zap.Int("id", id))
	return id, nil
}

func (o *OTPConn) isIDUnique(ctx context.Context, id int) (bool, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	o.logger.Info("Checking ID uniqueness", zap.Int("id", id))

	count, err := o.dbconn.CheckID(ctx, id)
	if err != nil {
		o.logger.Error("Failed to check ID in database", zap.Error(err))
		return false, err
	}

	isUnique := count == 0
	o.logger.Info("ID uniqueness check completed", zap.Bool("isUnique", isUnique))
	return isUnique, nil
}
