package auth_pin

import (
	"context"
	"errors"

	"github.com/aremxyplug-be/db"
	"github.com/aremxyplug-be/db/models"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

type PinConfig struct {
	dbConn db.Extras
	logger *zap.Logger
}

func NewPinConfig(logger *zap.Logger, store db.Extras) *PinConfig {
	return &PinConfig{
		dbConn: store,
		logger: logger,
	}
}

func (p *PinConfig) SavePin(ctx context.Context, pin models.UserPin) error {

	hashedPin, err := generatePin(pin.Pin)
	if err != nil {
		return err
	}

	pin.Pin = hashedPin
	if err := p.dbConn.SavePin(ctx, pin); err != nil {
		return err
	}

	return nil
}

var ErrIncorrectPin = errors.New("incorrect pin")

func (p *PinConfig) VerifyPin(ctx context.Context, userID, pin string) error {

	hashpin, err := p.dbConn.GetPin(ctx, userID)
	if err != nil {
		return err
	}

	if valid := comparePin(hashpin, pin); !valid {
		return ErrIncorrectPin
	}

	return nil
}

func (p *PinConfig) UpdatePin(ctx context.Context, userID string, newPin string) error {

	hashpin, err := generatePin(newPin)
	if err != nil {
		return err
	}

	pin := models.UserPin{
		UserID: userID,
		Pin:    hashpin,
	}

	if err := p.dbConn.UpdatePin(ctx, pin); err != nil {
		return err
	}

	return nil
}

func generatePin(pin string) (string, error) {
	pinByte, err := bcrypt.GenerateFromPassword([]byte(pin), 10)

	if err != nil {
		return "", err
	}

	return string(pinByte), nil
}

func comparePin(hashedPin, pin string) bool {

	err := bcrypt.CompareHashAndPassword([]byte(hashedPin), []byte(pin))
	return err == nil
}
