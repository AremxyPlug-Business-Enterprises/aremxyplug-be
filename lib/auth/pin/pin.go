package auth_pin

import (
	"log"

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

func (p *PinConfig) SavePin(pin models.UserPin) error {

	hashedPin, err := generatePin(pin.Pin)
	if err != nil {
		return err
	}

	pin.Pin = hashedPin
	if err := p.dbConn.SavePin(pin); err != nil {
		return err
	}

	return nil
}

func (p *PinConfig) VerifyPin(userID, pin string) (bool, error) {

	hashpin, err := p.dbConn.GetPin(userID)
	if err != nil {
		return false, err
	}

	if valid := comparePin(hashpin, pin); !valid {
		return false, err
	}

	return true, nil
}

func (p *PinConfig) UpdatePin(userID string, newPin string) error {

	hashpin, err := generatePin(newPin)
	if err != nil {
		return err
	}

	pin := models.UserPin{
		UserID: userID,
		Pin:    hashpin,
	}

	if err := p.dbConn.UpdatePin(pin); err != nil {
		return err
	}

	return nil
}

func generatePin(pin string) (string, error) {
	pinByte, err := bcrypt.GenerateFromPassword([]byte(pin), 10)

	//log.Println(string(pinByte))
	if err != nil {
		return "", err
	}

	return string(pinByte), nil
}

func comparePin(hashedPin, pin string) bool {

	log.Println(hashedPin)
	err := bcrypt.CompareHashAndPassword([]byte(hashedPin), []byte(pin))
	return err == nil
}
