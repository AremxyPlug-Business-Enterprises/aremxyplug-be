package otpgen

import (
	"log"
	"time"

	"github.com/aremxyplug-be/db"
	"github.com/aremxyplug-be/db/models"
	"github.com/pquerna/otp/totp"
)

type OTPConn struct {
	Dbconn db.DataStore
}

func NewOTP(DbConn db.DataStore) *OTPConn {
	return &OTPConn{
		Dbconn: DbConn,
	}
}

func (o *OTPConn) GenerateOTP(email string) (string, error) {
	key, err := totp.Generate(
		totp.GenerateOpts{
			Issuer:      "AremxyPlug",
			AccountName: email,
			Period:      300,
			Digits:      6,
		},
	)
	if err != nil {
		return "", err
	}

	log.Println(key.Secret(), key.URL())

	now := time.Now()

	data := models.OTP{
		Secret: key.Secret(),
		Email:  email,
	}

	if err := o.Dbconn.SaveOTP(data); err != nil {
		return "", err
	}

	otp, err := totp.GenerateCodeCustom(key.Secret(), now, totp.ValidateOpts{
		Period: 300,
		Digits: 6,
	})
	if err != nil {
		return "", err
	}

	return otp, nil

}

func (o *OTPConn) ValidateOTP(otp, email string) (bool, error) {

	// how do i get the email to search for the otp key associated with it?
	data, err := o.Dbconn.GetOTP(email)
	if err != nil {
		return false, err
	}

	now := time.Now()

	valid, err := totp.ValidateCustom(otp, data.Secret, now, totp.ValidateOpts{
		Period: 300,
		Digits: 6,
	})
	if err != nil {
		return false, err
	}
	if !valid {
		return false, err
	}

	return true, nil

}
