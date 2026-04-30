package models

import "time"

type ProviderRequestAudit struct {
	UserID          string                        `json:"user_id,omitempty" bson:"user_id,omitempty"`
	Product         string                        `json:"product" bson:"product"`
	ProviderName    string                        `json:"provider_name" bson:"provider_name"`
	Operation       string                        `json:"operation" bson:"operation"`
	Status          string                        `json:"status" bson:"status"`
	FailureType     string                        `json:"failure_type,omitempty" bson:"failure_type,omitempty"`
	TransactionID   string                        `json:"transaction_id,omitempty" bson:"transaction_id,omitempty"`
	OrderID         int                           `json:"order_id,omitempty" bson:"order_id,omitempty"`
	ReferenceNumber string                        `json:"reference_number,omitempty" bson:"reference_number,omitempty"`
	RequestID       string                        `json:"request_id,omitempty" bson:"request_id,omitempty"`
	PhoneNumber     string                        `json:"phone_number,omitempty" bson:"phone_number,omitempty"`
	Network         string                        `json:"network,omitempty" bson:"network,omitempty"`
	PlanID          int                           `json:"plan_id,omitempty" bson:"plan_id,omitempty"`
	PlanName        string                        `json:"plan_name,omitempty" bson:"plan_name,omitempty"`
	MeterNumber     string                        `json:"meter_number,omitempty" bson:"meter_number,omitempty"`
	SmartcardNumber string                        `json:"smartcard_number,omitempty" bson:"smartcard_number,omitempty"`
	DecoderType     string                        `json:"decoder_type,omitempty" bson:"decoder_type,omitempty"`
	PackageName     string                        `json:"package_name,omitempty" bson:"package_name,omitempty"`
	ExamType        string                        `json:"exam_type,omitempty" bson:"exam_type,omitempty"`
	Quantity        int                           `json:"quantity,omitempty" bson:"quantity,omitempty"`
	Request         ProviderAuditRequestSnapshot  `json:"request" bson:"request"`
	Response        ProviderAuditResponseSnapshot `json:"response" bson:"response"`
	ErrorMessage    string                        `json:"error_message,omitempty" bson:"error_message,omitempty"`
	ProviderStatus  string                        `json:"provider_status,omitempty" bson:"provider_status,omitempty"`
	ProviderMessage string                        `json:"provider_message,omitempty" bson:"provider_message,omitempty"`
	Metadata        map[string]interface{}        `json:"metadata,omitempty" bson:"metadata,omitempty"`
	CreatedAt       time.Time                     `json:"created_at" bson:"created_at"`
}

type ProviderAuditRequestSnapshot struct {
	Method  string            `json:"method" bson:"method"`
	URL     string            `json:"url" bson:"url"`
	Headers map[string]string `json:"headers,omitempty" bson:"headers,omitempty"`
	Query   map[string]string `json:"query,omitempty" bson:"query,omitempty"`
	Form    map[string]string `json:"form,omitempty" bson:"form,omitempty"`
	Body    string            `json:"body,omitempty" bson:"body,omitempty"`
}

type ProviderAuditResponseSnapshot struct {
	StatusCode int                    `json:"status_code,omitempty" bson:"status_code,omitempty"`
	Headers    map[string]string      `json:"headers,omitempty" bson:"headers,omitempty"`
	Body       string                 `json:"body,omitempty" bson:"body,omitempty"`
	Decoded    map[string]interface{} `json:"decoded,omitempty" bson:"decoded,omitempty"`
}
