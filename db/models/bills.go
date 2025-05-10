package models

type TvInfo struct {
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
type TvAPI struct {
	Code      string     `json:"code"`
	Content   Tv_Content `json:"content"`
	Date      string     `json:"transaction_date"` // Direct string match for ISO datetime
	RequestID string     `json:"requestId"`
	Response  string     `json:"response_description"`
}

type Tv_Content struct {
	Transactions Transactions_Details `json:"transactions"` // Fixed spelling to match JSON
}

type Transactions_Details struct {
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

type BillResult struct {
	DecoderType   string `json:"decoder_type"`
	Package       string `json:"package"`
	IucNumber     string `json:"iuc_number"`
	Phone         string `json:"phone"`
	Email         string `json:"email"`
	Name          string `json:"name"`
	Amount        int    `json:"amount"`
	Product       string `json:"product"`
	Description   string `json:"description"`
	OrderID       int    `json:"order_id"`
	TranscationID string `json:"transcation_id"`
	RequestID     string `json:"request_id"`
}
