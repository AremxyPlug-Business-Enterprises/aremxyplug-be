package termii

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
	PinID     string `json:"pinId"`
	To        string `json:"to"`
	SmsStatus string `json:"smsStatus"`
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
