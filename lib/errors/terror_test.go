package errors

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTerror_Error(t *testing.T) {
	var tests = []struct {
		name   string
		terror *Terror
		want   string
	}{
		{
			name: "Test only mandatory attrs",
			terror: NewTerror(
				7001,
				"InvalidPhoneNumberException",
				"Provided phone number is already attached to Roava account",
				"This phone number is already attached to a Roava account. Kindly recheck the phone number or logon to continue",
			),
			want: "{\"error\":{\"code\":7001,\"type\":\"InvalidPhoneNumberException\",\"message\":\"Provided phone number is already attached to Roava account\",\"detail\":\"This phone number is already attached to a Roava account. Kindly recheck the phone number or logon to continue\"}}",
		},
		{
			name: "Test with status",
			terror: NewTerror(
				7001,
				"InvalidPhoneNumberException",
				"Provided phone number is already attached to Roava account",
				"This phone number is already attached to a Roava account. Kindly recheck the phone number or logon to continue",
				WithStatus(http.StatusOK),
			),
			want: "{\"error\":{\"code\":7001,\"type\":\"InvalidPhoneNumberException\",\"message\":\"Provided phone number is already attached to Roava account\",\"status\":200,\"detail\":\"This phone number is already attached to a Roava account. Kindly recheck the phone number or logon to continue\"}}",
		},
		{
			name: "Test with instance",
			terror: NewTerror(
				7001,
				"InvalidPhoneNumberException",
				"Provided phone number is already attached to Roava account",
				"This phone number is already attached to a Roava account. Kindly recheck the phone number or logon to continue",
				WithInstance("srv.onboarding"),
			),
			want: "{\"error\":{\"code\":7001,\"type\":\"InvalidPhoneNumberException\",\"message\":\"Provided phone number is already attached to Roava account\",\"detail\":\"This phone number is already attached to a Roava account. Kindly recheck the phone number or logon to continue\",\"instance\":\"srv.onboarding\"}}",
		},
		{
			name: "Test with trace_id",
			terror: NewTerror(
				7001,
				"InvalidPhoneNumberException",
				"Provided phone number is already attached to Roava account",
				"This phone number is already attached to a Roava account. Kindly recheck the phone number or logon to continue",
				WithTraceID("ckadzrzx4000001l8cnh85jdr"),
			),
			want: "{\"error\":{\"code\":7001,\"type\":\"InvalidPhoneNumberException\",\"message\":\"Provided phone number is already attached to Roava account\",\"detail\":\"This phone number is already attached to a Roava account. Kindly recheck the phone number or logon to continue\",\"trace_id\":\"ckadzrzx4000001l8cnh85jdr\"}}",
		},
		{
			name: "Test with help",
			terror: NewTerror(
				7001,
				"InvalidPhoneNumberException",
				"Provided phone number is already attached to Roava account",
				"This phone number is already attached to a Roava account. Kindly recheck the phone number or logon to continue",
				WithHelp("http://somehelpfulwebsite.com"),
			),
			want: "{\"error\":{\"code\":7001,\"type\":\"InvalidPhoneNumberException\",\"message\":\"Provided phone number is already attached to Roava account\",\"detail\":\"This phone number is already attached to a Roava account. Kindly recheck the phone number or logon to continue\",\"help\":\"http://somehelpfulwebsite.com\"}}",
		},
		{
			name: "Test with all attrs",
			terror: NewTerror(
				7001,
				"InvalidPhoneNumberException",
				"Provided phone number is already attached to Roava account",
				"This phone number is already attached to a Roava account. Kindly recheck the phone number or logon to continue",
				WithStatus(http.StatusOK),
				WithInstance("srv.onboarding"),
				WithTraceID("ckadzrzx4000001l8cnh85jdr"),
				WithHelp("http://somehelpfulwebsite.com"),
			),
			want: "{\"error\":{\"code\":7001,\"type\":\"InvalidPhoneNumberException\",\"message\":\"Provided phone number is already attached to Roava account\",\"status\":200,\"detail\":\"This phone number is already attached to a Roava account. Kindly recheck the phone number or logon to continue\",\"trace_id\":\"ckadzrzx4000001l8cnh85jdr\",\"instance\":\"srv.onboarding\",\"help\":\"http://somehelpfulwebsite.com\"}}",
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.want, testCase.terror.Error())
		})
	}
}

func TestNewTerrorFromJSONString(t *testing.T) {
	var tests = []struct {
		name       string
		jsonString string
		want       *Terror
		error      bool
	}{
		{
			name:       "Test created correctly",
			jsonString: "{\"error\":{\"code\":7001,\"type\":\"InvalidPhoneNumberException\",\"message\":\"Provided phone number is already attached to Roava account\",\"detail\":\"This phone number is already attached to a Roava account. Kindly recheck the phone number or logon to continue\"}}",
			want: &Terror{
				code:      7001,
				errorType: "InvalidPhoneNumberException",
				message:   "Provided phone number is already attached to Roava account",
				detail:    "This phone number is already attached to a Roava account. Kindly recheck the phone number or logon to continue",
			},
			error: false,
		},
		{
			name:       "Test created correctly all attributes",
			jsonString: "{\"error\":{\"code\":7001,\"type\":\"InvalidPhoneNumberException\",\"message\":\"Provided phone number is already attached to Roava account\",\"status\":200,\"detail\":\"This phone number is already attached to a Roava account. Kindly recheck the phone number or logon to continue\",\"trace_id\":\"ckadzrzx4000001l8cnh85jdr\",\"instance\":\"srv.onboarding\",\"help\":\"http://somehelpfulwebsite.com\"}}",
			want: &Terror{
				code:      7001,
				errorType: "InvalidPhoneNumberException",
				message:   "Provided phone number is already attached to Roava account",
				status:    200,
				detail:    "This phone number is already attached to a Roava account. Kindly recheck the phone number or logon to continue",
				traceID:   "ckadzrzx4000001l8cnh85jdr",
				instance:  "srv.onboarding",
				help:      "http://somehelpfulwebsite.com",
			},
			error: false,
		},
		{
			name:       "Test error unmarshalling json",
			jsonString: "bad json structure",
			error:      true,
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			terror, err := NewTerrorFromJSONString(testCase.jsonString)

			if testCase.error {
				assert.Error(t, err)
				return
			}
			assert.Equal(t, testCase.want, terror)
			assert.NoError(t, err)
		})
	}
}
