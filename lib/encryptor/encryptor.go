package encryptor

import (
	"golang.org/x/crypto/bcrypt"
)

type Encryptor interface {
	ComparePasscode(passcode, hashedPasscode string) bool
	GenerateFromPassword(password string) ([]byte, error)
}

type encryptor struct {
}

func NewEncryptor() Encryptor {
	return &encryptor{}
}

func (e *encryptor) ComparePasscode(passcode, hashedPasscode string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPasscode), []byte(passcode))
	return err == nil
}

func (e *encryptor) GenerateFromPassword(password string) ([]byte, error) {
	return bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
}
