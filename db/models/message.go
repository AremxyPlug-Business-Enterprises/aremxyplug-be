package models

// Message model (Messages managed by ROAVA)
type Message struct {
	ID          string            `json:"id" bson:"id"`
	CustomerID  string            `json:"customer_id" bson:"customer_id"`
	AccountID   string            `json:"account_id" bson:"account_id"`
	Target      string            `json:"target" bson:"target"`
	Type        MessageType       `json:"type" bson:"type"`
	Title       string            `json:"title" bson:"title"`
	Body        string            `json:"body" bson:"body"`
	TemplateID  string            `json:"template_id" bson:"template_id"`
	DataMap     map[string]string `json:"data_map" bson:"data_map"`
	Attachments []Attachment      `json:"attachments" bson:"attachments"`
	Ts          int64             `json:"ts" bson:"ts"`
}

type Attachment struct {
	Name        string `json:"name" bson:"name"`
	Content     string `json:"content" bson:"content"`
	ContentType string `json:"content_type" bson:"content_type"`
}

// MessageType enum type
type MessageType string

const (
	PUSH_MESSAGE_TYPE  MessageType = "PUSH"
	EMAIL_MESSAGE_TYPE MessageType = "EMAIL"
	SMS_MESSAGE_TYPE   MessageType = "SMS"
)
