package termii

import (
	"encoding/json"
	"fmt"
)

type smsRequest struct {
	APIKey         string `json:"api_key"`
	MessageType    string `json:"message_type"`
	To             string `json:"to"`
	From           string `json:"from"`
	Channel        string `json:"channel"`
	PinAttempts    int    `json:"pin_attempts"`
	PinTimeToLive  int    `json:"pin_time_to_live"`
	PinLength      int    `json:"pin_length"`
	PinPlaceholder string `json:"pin_placeholder"`
	MessageText    string `json:"message_text"`
	PinType        string `json:"pin_type"`
}

type smsClientResponse struct {
	PinID      string `json:"pinId"`
	To         string `json:"to"`
	SmsStatus  string `json:"smsStatus"`
	StatusCode string `json:"status"`
}

type verifyTokenRequest struct {
	APIKey string `json:"api_key"`
	PinID  string `json:"pin_id"`
	Pin    string `json:"pin"`
}

type verifyTokenResponse struct {
	PinID    string `json:"pinId"`
	Verified bool   `json:"verified"`
	Msisdn   string `json:"msisdn"`
}

// UnmarshalJSON custom unmarshaller for verifyTokenResponse
func (v *verifyTokenResponse) UnmarshalJSON(data []byte) error {
	type Alias verifyTokenResponse
	aux := &struct {
		Verified interface{} `json:"verified"`
		*Alias
	}{
		Alias: (*Alias)(v),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	switch value := aux.Verified.(type) {
	case bool:
		v.Verified = value
	case string:
		switch value {
		case "Expired":
			v.Verified = false

		default:
			return fmt.Errorf("unexpected value for string: %v", value)
		}

	default:
		return fmt.Errorf("unexpected type for verified field: %T", value)
	}

	return nil
}

type whatsAppRequest struct {
	APIKey  string `json:"api_key"`
	To      string `json:"to"`
	From    string `json:"from"`
	SMS     string `json:"sms"`
	Channel string `json:"channel"`
	Type    string `json:"type"`
}

type whatsAppResponse struct {
	Code         string  `json:"code"`
	Balance      float64 `json:"balance"`
	MessageID    string  `json:"message_id"`
	Message      string  `json:"message"`
	User         string  `json:"user"`
	MessageIDStr string  `json:"message_id_str"`
}

type kudiSMSWhatsAppResponse struct {
	Status    string  `json:"status"`
	ErrorCode string  `json:"error_code"`
	Cost      float64 `json:"cost"`
	Data      string  `json:"data"`
	Msg       string  `json:"msg"`
	Balance   float64 `json:"balance"`
}
