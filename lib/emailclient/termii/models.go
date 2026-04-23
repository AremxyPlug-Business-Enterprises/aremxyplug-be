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
