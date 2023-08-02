package errorvalues

import (
	"fmt"
	"github.com/aremxyplug-be/lib/errors"
)

// Error codes
const (
	DatabaseError                     = 7401
	DatabaseNotFoundError             = 7402
	InvalidAuthenticationError        = 7403
	PulsarError                       = 7404
	InvalidRequestErr                 = 7405
	CustomerNotFound                  = 7406
	InternalServerError               = 7407
	SamePhoneNumberError              = 7408
	DuplicateCustomerPhoneNumberError = 7409
	InvalidPhoneNumberError           = 7410
	DuplicatedCustomerEmailError      = 7411
	AuthenthicationFailedErr          = 7412
	InvalidTokenErr                   = 7413
)

var (
	errorTypes = map[int]string{
		DatabaseError:                     "DatabaseError",
		PulsarError:                       "PulsarError",
		InvalidRequestErr:                 "InvalidRequestErr",
		CustomerNotFound:                  "CustomerNotFound",
		DatabaseNotFoundError:             "DatabaseNotFoundError",
		InvalidAuthenticationError:        "InvalidAuthenticationError",
		InternalServerError:               "InternalServerError",
		SamePhoneNumberError:              "SamePhoneNumberError",
		DuplicateCustomerPhoneNumberError: "DuplicateCustomerPhoneNumberError",
		InvalidPhoneNumberError:           "InvalidPhoneNumberError",
		DuplicatedCustomerEmailError:      "DuplicatedCustomerEmailError",
	}

	errorMessages = map[int]string{
		InvalidRequestErr:                 "invalid request error, failed to parse event",
		DatabaseError:                     "failed to handle request at this time due to technical issues. Please retry",
		PulsarError:                       "failed to handle request at this time due to technical issues. Please retry",
		CustomerNotFound:                  "customer's account not found",
		DatabaseNotFoundError:             "model not found.",
		InvalidAuthenticationError:        "invalid authentication",
		InternalServerError:               "failed to handle request at this time due to technical issues. Please retry",
		SamePhoneNumberError:              "you have used the same phone number as the one that is currently on your account, please use a different phone number to update to a new one",
		DuplicateCustomerPhoneNumberError: "phone number is already registered on roava, please input a different phone number",
		InvalidPhoneNumberError:           "please enter a valid phone number",
		DuplicatedCustomerEmailError:      "email is registered with an existing rova customer",
	}
)

func Type(code int) string {
	if value, ok := errorTypes[code]; ok {
		return value
	}
	return "UnKnownError"
}

func Message(code int) string {
	if value, ok := errorMessages[code]; ok {
		return value
	}
	return "unknown"
}

func Format(code int, err error) error {
	return errors.NewTerror(code, Type(code), Message(code), fmt.Sprintf("%s: %v", Message(code), err))
}
