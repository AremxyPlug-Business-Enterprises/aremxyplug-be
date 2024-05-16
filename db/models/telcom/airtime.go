package telcom

type AirtimeInfo struct {
	Network     string `json:"network"`
	Amount      string `json:"amount"`
	Phone_no    string `json:"mobileno"`
	Product     string `json:"product"`
	Recipient   string `json:"recipient,omitempty"`
	AirtimeType string `json:"airtime_type"`
}

type AirtimeApiResponse struct {
	Success_Response string  `json:"success"`
	Message          string  `json:"message"`
	Network          string  `json:"network"`
	Phone_no         string  `json:"mobileno"`
	Amount           int     `json:"airtimeamount"`
	Charged          float64 `json:"amountcharged"`
	Status           string  `json:"status"`
	Date             string  `json:"transaction_date"`
	Reference        string  `json:"reference_no"`
}

type AirtimeResponse struct {
	Status          string `json:"status" bson:"status"`
	Network         string `json:"network" bson:"network"`
	Amount          string `json:"amount" bson:"amount"`
	Phone_no        string `json:"phone_no" bson:"phone_no"`
	Name            string `json:"name" bson:"name"`
	Product         string `json:"product" bson:"product"`
	Recipient       string `json:"recipient,omitempty" bson:"recipient,omitempty"`
	OrderID         int    `json:"order_id" bson:"order_id"`
	Description     string `json:"description" bson:"description"`
	TransactionID   string `json:"transaction_id" bson:"transaction_id"`
	ReferenceNumber string `json:"reference_number" bson:"reference_number"`
}
