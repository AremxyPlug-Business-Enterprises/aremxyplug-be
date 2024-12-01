package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
)

const secret = "your_secret_token"

func VerifySignature(r *http.Request) bool {
	signature := r.Header.Get("X-Signature")
	if signature == "" {
		return false
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return false
	}
	defer r.Body.Close()

	h := hmac.New(sha256.New, []byte(secret))
	h.Write(body)
	expectedSignature := hex.EncodeToString(h.Sum(nil))

	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}
