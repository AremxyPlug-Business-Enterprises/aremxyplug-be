package config

import (
	"errors"
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Secrets struct {
	RefreshTokenDuration int    `json:"REFRESH_TOKEN_DURATION"`
	AuthTokenDuration    int    `json:"AUTH_TOKEN_DURATION"`
	JWTPublicKey         string `json:"JWT_PUBLIC_KEY"`
	JWTPrivateKey        string `json:"JWT_PRIVATE_KEY"`
	DbName               string `json:"DB_NAME"`
	MongdbUrl            string `json:"MONGODB_URI"`
	AppPort              string `json:"PORT"`
	PlatformEmail        string `json:"PLATFORM_EMAIL"`
	PostmarkKey          string `json:"POSTMARK_KEY"`
	accountsid           string `json:"TWILIO_ACCOUNT_SID"`
	authtoken            string `json:"TWILIO_AUTH_TOKEN"`
	ServiceID            string `json:"TWILIO_SERVICES_ID"`
}

var ss Secrets

func init() {
	if os.Getenv("APP_ENV") != "production" {
		err := godotenv.Load(".env")
		if err != nil {
			log.Fatalf("Error loading .env file: \n %v", err)
		}
	}

	ss = Secrets{}
	ss.RefreshTokenDuration, _ = getenvInt("REFRESH_TOKEN_DURATION")
	ss.AuthTokenDuration, _ = getenvInt("AUTH_TOKEN_DURATION")
	ss.JWTPublicKey = os.Getenv("JWT_PUBLIC_KEY")
	ss.JWTPrivateKey = os.Getenv("JWT_PRIVATE_KEY")
	ss.DbName = os.Getenv("DB_NAME")
	ss.MongdbUrl = os.Getenv("MONGODB_URL")
	ss.PlatformEmail = os.Getenv("PLATFORM_EMAIL")
	ss.PostmarkKey = os.Getenv("POSTMARK_KEY")
	ss.AppPort = os.Getenv("PORT")

	if ss.AppPort = os.Getenv("PORT"); ss.AppPort == "" {
		ss.AppPort = "8080"
	}

}

// GetSecrets is used to get value from the Secrets runtime.
func GetSecrets() *Secrets {
	return &ss
}

var ErrEnvVarEmpty = errors.New("getenv: environment variable empty")

func getenvStr(key string) (string, error) {
	v := os.Getenv(key)
	if v == "" {
		return v, ErrEnvVarEmpty
	}
	return v, nil
}

func getenvInt(key string) (int, error) {
	s, err := getenvStr(key)
	if err != nil {
		return 0, err
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return 0, err
	}
	return v, nil
}
