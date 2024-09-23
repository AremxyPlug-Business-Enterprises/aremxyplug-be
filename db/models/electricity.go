package models

type ElectricInfo struct {
	DiscoType  string `json:"disco_type"` // Name of service to buy
	Meter_No   string `json:"meter_no"`   // meter number
	Meter_Type string `json:"meter_type"` // meter type
	Amount     int    `json:"amount"`
	Phone      string `json:"phone"`
	Email      string `json:"email"`
	RequestID  string `json:"request_id"`
}

type ElectricAPI struct {
	Code            string          `json:"code"`
	Contents        Content         `json:"content"`
	TransactionDate TransactionDate `json:"transaction_date"`
	RequestID       string          `json:"requestId"`
	Amount          string          `json:"amount"`
	Purchased_Token string          `json:"purchased_code"`
}

type Content struct {
	Transactions TransactionDetails `json:"transactions"`
}

type TransactionDetails struct {
	Status        string  `json:"status"`
	Product_Name  string  `json:"product_name"` // map to description, split end to bill generated
	Meter_No      string  `json:"unique_element"`
	Unit_Price    float64 `json:"unit_price"`
	Commission    float64 `json:"commission"`
	Phone         string  `json:"phone"`
	Type          string  `json:"type"`
	TransactionID string  `json:"transactionId"`
	Email         string  `json:"email"`
}

type TransactionDate struct {
	Date string `json:"date"`
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
	Amount        string `json:"amount"`
	DiscoType     string `json:"disco_type" bson:"DiscoType"`
	MeterType     string `json:"meter_type" bson:"meter_type"` // Prepaid
	Name          string `json:"name" bson:"name"`
	MeterNumber   string `json:"meter_number" bson:"meter_number"`
	Phone         string `json:"phone" bson:"phone"`
	Email         string `json:"email" bson:"email"`
	Product       string `json:"product" bson:"product"`
	Description   string `json:"description" bson:"description"` // append serviceID and variation code.
	BillGenerated string `json:"bill_generated" bson:"bill_generated"`
	OrderID       int    `json:"order_id" bson:"order_id"`
	TransactionID string `json:"transaction_id" bson:"transaction_id"`
	RequestID     string `json:"request_id" bson:"request_ID"`
}
