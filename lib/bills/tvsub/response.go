package tvsub

type serverResponse struct {
	Code    string  `json:"code"`
	Content content `json:"content"`
}

type content struct {
	CustomerName      string            `json:"Customer_Name"`
	Status            string            `json:"Status"`
	DueDate           string            `json:"Due_Date"`
	CustomerNumber    string            `json:"Customer_Number"`
	CustomerType      string            `json:"Customer_Type"`
	CommissionDetails commissionDetails `json:"commission_details"`
}

type commissionDetails struct {
	Amount          *float64 `json:"amount"` // Use pointer to handle null values
	Rate            string   `json:"rate"`
	RateType        string   `json:"rate_type"`
	ComputationType string   `json:"computation_type"`
}

type verifyResponse struct {
	Name  string `json:"name"`
	Phone string `json:"phone"`
}

type TvInfo struct {
	UserID           string
	DecoderType      string `json:"decoder_type"`
	SmartCard_Number string `json:"iuc_number"`
	Package          string `json:"package"`
	Email            string `json:"email"`
	Amount           int    `json:"amount"`
	Phone            string `json:"phone"`
	SubType          string `json:"sub_type"`
	RequestID        string `json:"request_id"`
	Name             string
}
type tvAPI struct {
	Code          string     `json:"code"`
	Content       tv_Content `json:"content"`
	Date          string     `json:"transaction_date"` // Direct string match for ISO datetime
	RequestID     string     `json:"requestId"`
	Response      string     `json:"response_description"`
	PurchasedCode string     `json:"purchased_code"`
}

type tv_Content struct {
	Transactions transactions_Details `json:"transactions"` // Fixed spelling to match JSON
}

type transactions_Details struct {
	Status        string  `json:"status"`
	Product_Desc  string  `json:"product_name"`
	Unit_Price    float64 `json:"unit_price,string"` // Handle string->float conversion
	Commission    float64 `json:"commission"`
	Email         string  `json:"email"`
	Phone         string  `json:"phone"`
	Amount        float64 `json:"amount,string"` // Handle string->float conversion
	TransactionID string  `json:"transactionId"`
	Type          string  `json:"type"`
}
