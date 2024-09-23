package webhook

import (
	"io"
	"log"
	"net/http"
)

func handlewebhook(r *http.Request) ([]byte, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {

	}
	log.Printf("%s", string(body))

	return body, nil
}
