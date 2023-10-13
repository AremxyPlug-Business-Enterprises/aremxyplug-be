package models

type ElectricInfo struct {
	ServiceID     string `json:"service_id"`     // Name of service to buy
	BillersCode   string `json:"billers_code"`   // meter number
	VariationCode string `json:"variation_code"` // meter type
	Amount        int    `json:"amount"`
	Phone         int    `json:"phone"`
	RequestID     string `json:"request_id"`
}

type ElectricAPI struct {
	Code            int             `json:"code"`
	Contents        Content         `json:"content"`
	TransactionDate TransactionDate `json:"transaction_date"`
	RequestID       string          `json:"requestId"`
	Amount          float64         `json:"amount"`
}

type Content struct {
	Transactions TransactionDetails `json:"transactions"`
}

type TransactionDetails struct {
	Status        string  `json:"status"`
	Product_Name  string  `json:"product_name"` // map to description, split end to bill generated
	Unit_Price    float64 `json:"unit_price"`
	Commission    float64 `json:"commission"`
	Phone         string  `json:"phone"`
	TransactionID string  `json:"transactionId"`
	Email         string  `json:"email"`
}

type TransactionDate struct {
	Date string `json:"date"`
}

type ElectricResult struct {
	DiscoType     string `json:"disco_type"`
	MeterType     string `json:"meter_type"` // Prepaid
	Name          string `json:"name"`
	MeterNumber   int    `json:"meter_number"`
	Phone         string `json:"phone"`
	Email         string `json:"email"`
	Product       string `json:"product"`
	Description   string `json:"description"` // append serviceID and variation code.
	BillGenerated string `json:"bill_generated"`
	OrderID       int    `json:"order_id"`
	TransactionID string `json:"transaction_id"`
}
