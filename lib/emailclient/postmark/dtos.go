package postmark

type EmailTemplateAttachment struct {
	Name        string `json:"Name"`
	Content     string `json:"Content"`
	ContentType string `json:"ContentType"`
}

// EmailWithTemplateRequest payload definition
type EmailWithTemplateRequest struct {
	TemplateAlias string                 `json:"TemplateAlias"`
	TemplateModel map[string]interface{} `json:"TemplateModel"`
	InlineCSS     bool                   `json:"InlineCss,omitempty"`
	From          string                 `json:"From"`
	To            string                 `json:"To"`
	Cc            string                 `json:"Cc,omitempty"`
	Bcc           string                 `json:"Bcc,omitempty"`
	Tag           string                 `json:"Tag,omitempty"`
	ReplyTo       string                 `json:"ReplyTo,omitempty"`
	Headers       []struct {
		Name  string `json:"Name"`
		Value string `json:"Value"`
	} `json:"Headers,omitempty"`
	TrackOpens  *bool                     `json:"TrackOpens,omitempty"`
	TrackLinks  string                    `json:"TrackLinks,omitempty"`
	Attachments []EmailTemplateAttachment `json:"Attachments,omitempty"`
	Metadata    struct {
		Color    string `json:"color"`
		ClientID string `json:"client-id"`
	} `json:"Metadata,omitempty"`
	MessageStream string `json:"MessageStream,omitempty"`
}

// EmailWithTemplateResponse payload definition
type EmailWithTemplateResponse struct {
	To          string `json:"To"`
	SubmittedAt string `json:"SubmittedAt"`
	MessageID   string `json:"MessageID"`
	ErrorCode   int    `json:"ErrorCode"`
	Message     string `json:"Message"`
}

// ErrorResponse payload definition
type ErrorResponse struct {
	ErrorCode int    `json:"ErrorCode"`
	Message   string `json:"Message"`
}
