package referral

import (
	"math/rand"
	"time"

	"github.com/aremxyplug-be/db"
)

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

type RefConfig struct {
	db db.Extras
}

func NewRefConfig(store db.Extras) *RefConfig {
	return &RefConfig{
		db: store,
	}
}

func (r *RefConfig) CreateReferral(userID string) (string, error) {
	code := generateReferralCode(6)

	err := r.db.CreateUserReferral(userID, code)
	if err != nil {
		return "", err
	}

	return code, nil
}

func (r *RefConfig) GetReferral(userID string) (string, error) {

	code, err := r.db.GetReferral(userID)
	if err != nil {
		return "", err
	}

	return code, nil
}

func (r *RefConfig) UpdateReferralCount(userID, referralCode string) error {

	err := r.db.UpdateReferralCount(referralCode)
	if err != nil {
		return err
	}

	return nil
}

func generateReferralCode(length int) string {

	var seededRand *rand.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))

	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}
