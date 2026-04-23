package termii

type emailRequest struct {
	EmailAddress  string `json:"email_address"`
	Code          string `json:"code"`
	APIKey        string `json:"api_key"`
	EmailConfigID string `json:"email_configuration_id"`
}

type emailResponse struct {
	Code      string `json:"code"`
	MessageID string `json:"message_id"`
	Message   string `json:"message"`
	Balance   int    `json:"balance"`
	User      string `json:"user"`
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
