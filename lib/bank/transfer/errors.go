package transfer

import (
	"errors"
)

var (
	ErrAccountValidationFailed    = errors.New("failed to verify account")
	ErrCounterpartyCreationFailed = errors.New("failed to create counterparty")
	ErrAPIConnectionFailed        = errors.New("error connecting to API server")
	ErrCreatingHTTPRequest        = errors.New("error creating HTTP request")
	ErrGeneratingOrderID          = errors.New("error generating order_id")
	ErrReadingRequestBody         = errors.New("error reading request body")
)

func JSONError(err error) error {
	return errors.New("error marshalling/decoding JSON representation: " + err.Error())
}

func DBConnectionError(err error) error {
	return errors.New("error while communicting with DB: " + err.Error())
}
