package models

import "time"

type ElectricInfo struct {
	DiscoType  string `json:"disco_type"` // Name of service to buy
	Meter_No   string `json:"meter_no"`   // meter number
	Meter_Type string `json:"meter_type"` // meter type
	Amount     int    `json:"amount"`
	Phone      string `json:"phone"`
	Email      string `json:"email"`
	RequestID  string `json:"request_id"`
	Name       string
}

type ElectricAPI struct {
	Code                 string  `json:"code"`
	Content              Content `json:"content"`
	ResponseDescription  string  `json:"response_description"`
	RequestID            string  `json:"requestId"`
	Amount               float64 `json:"amount"`
	TransactionDate      string  `json:"transaction_date"`
	PurchasedToken       string  `json:"purchased_code"`
	CustomerName         string  `json:"customerName"`
	CustomerAddress      string  `json:"customerAddress"`
	MeterNumber          string  `json:"meterNumber"`
	Token                string  `json:"token"`
	TokenAmount          float64 `json:"tokenAmount"`
	ExchangeReference    string  `json:"exchangeReference"`
	ResetToken           string  `json:"resetToken"`
	ConfigureToken       string  `json:"configureToken"`
	Units                string  `json:"units"`
	FixChargeAmount      float64 `json:"fixChargeAmount"`
	Tariff               string  `json:"tariff"`
	TaxAmount            float64 `json:"taxAmount"`
	DebtAmount           float64 `json:"debtAmount"`
	Kct1                 string  `json:"kct1"`
	Kct2                 string  `json:"kct2"`
	Penalty              float64 `json:"penalty"`
	CostOfUnit           float64 `json:"costOfUnit"`
	Announcement         string  `json:"announcement"`
	MeterCost            float64 `json:"meterCost"`
	CurrentCharge        float64 `json:"currentCharge"`
	LossOfRevenue        float64 `json:"lossOfRevenue"`
	TariffBaseRate       float64 `json:"tariffBaseRate"`
	InstallationFee      float64 `json:"installationFee"`
	ReconnectionFee      float64 `json:"reconnectionFee"`
	MeterServiceCharge   float64 `json:"meterServiceCharge"`
	AdministrativeCharge float64 `json:"administrativeCharge"`
}

type Content struct {
	Transactions TransactionDetails `json:"transactions"`
}

type TransactionDetails struct {
	Status              string            `json:"status"`
	ProductName         string            `json:"product_name"`
	UniqueElement       string            `json:"unique_element"`
	UnitPrice           string            `json:"unit_price"`
	Quantity            float64           `json:"quantity"`
	ServiceVerification interface{}       `json:"service_verification"`
	Channel             string            `json:"channel"`
	Commission          float64           `json:"commission"`
	TotalAmount         float64           `json:"total_amount"`
	Discount            interface{}       `json:"discount"`
	Type                string            `json:"type"`
	Email               string            `json:"email"`
	Phone               string            `json:"phone"`
	Name                interface{}       `json:"name"`
	ConvinienceFee      float64           `json:"convinience_fee"` // Note JSON typo
	Amount              string            `json:"amount"`
	Platform            string            `json:"platform"`
	Method              string            `json:"method"`
	TransactionID       string            `json:"transactionId"`
	CommissionDetails   CommissionDetails `json:"commission_details"`
}

type CommissionDetails struct {
	Amount          float64 `json:"amount"`
	Rate            string  `json:"rate"`
	RateType        string  `json:"rate_type"`
	ComputationType string  `json:"computation_type"`
}

type VerifyMeterResponse struct {
	Code    string        `json:"code"`
	Content verifyContent `json:"content"`
}

type verifyContent struct {
	Name         string `json:"name"`
	Meter_Number string `json:"meter_number"`
	Err          string `json:"error,omitempty"`
}

type ElectricResult struct {
	Amount         string    `json:"amount"`
	DiscoType      string    `json:"disco_type" bson:"DiscoType"`
	MeterType      string    `json:"meter_type" bson:"meter_type"` // Prepaid
	Name           string    `json:"name" bson:"name"`
	MeterNumber    string    `json:"meter_number" bson:"meter_number"`
	Phone          string    `json:"phone" bson:"phone"`
	Email          string    `json:"email" bson:"email"`
	Product        string    `json:"product" bson:"product"`
	Description    string    `json:"description" bson:"description"` // append serviceID and variation code.
	BillGenerated  string    `json:"bill_generated" bson:"bill_generated"`
	OrderID        int       `json:"order_id" bson:"order_id"`
	TransactionID  string    `json:"transaction_id" bson:"transaction_id"`
	ReferenceNuber string    `json:"reference_number" bson:"reference_number"`
	RequestID      string    `json:"request_id" bson:"request_ID"`
	CreatedAt      time.Time `json:"created_at" bson:"created_at"`
}
