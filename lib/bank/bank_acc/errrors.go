package bankacc

import "errors"

var (
	ErrAccountValidationFailed    = errors.New("failed to verify account")
	ErrCounterpartyCreationFailed = errors.New("failed to create counterparty")
	ErrAPIConnectionFailed        = errors.New("error connecting to API server")
	ErrCreatingHTTPRequest        = errors.New("error creating HTTP request")
	ErrCreatingDepositAccount     = errors.New("error creating deposit account")
	ErrWritingToENV               = errors.New("error writing to .env file")
	ErrSettingENV                 = errors.New("error setting environment variable")
)

func JSONError(err error) error {
	return errors.New("error marshalling/decoding JSON representation\n" + err.Error())
}

func DBConnectionError(err error) error {
	return errors.New("error while communicting with DB\n" + err.Error())
}
