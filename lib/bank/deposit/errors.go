package deposit

import "errors"

var (
	ErrNewRequestFailed           = errors.New("failed to initialize new request")
	ErrCounterpartyCreationFailed = errors.New("failed to create counterparty")
	ErrAPIConnectionFailed        = errors.New("error connecting to API server")
	ErrCreatingHTTPRequest        = errors.New("error creating HTTP request")
)

func JSONError(err error) error {
	return errors.New("error marshalling/decoding JSON representation" + err.Error())
}

func DBConnectionError(err error) error {
	return errors.New("error while communicting with DB" + err.Error())
}
