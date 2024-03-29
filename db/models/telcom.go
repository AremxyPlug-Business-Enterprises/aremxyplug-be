package models

type DataInfo struct {
	Network       int    `json:"network"`
	Network_id    int    `json:"newtork_id"`
	Plan          int    `json:"plan"`
	Plan_id       string `json:"plan_id"`
	Mobile_Num    string `json:"mobile_number"`
	Ported_number bool   `json:"Ported_number"`
	Name          string `json:"name"`
}

type DataResult struct {
	OrderID         int    `json:"order_id" bson:"order_id"`
	TransactionID   string `json:"transaction_id" bson:"transaction_id"`
	ReferenceNumber string `json:"reference_number" bson:"reference_number"`
	Network         string `json:"network" bson:"network"`
	Username        string `json:"username" bson:"username"`
	PlanName        string `json:"plan_name" bson:"plan_name"`
	Plan_Amount     string `json:"plan_amount" bson:"plan_amount"`
	Status          string `json:"Status" bson:"status"`
	Name            string `json:"Name" bson:"name"`
	Phone_Number    string `json:"Phone_Number" bson:"phone_number"`
	CreatedAt       string `json:"CreatedAt" bson:"created_at"`
	ApiID           int    `bson:"apiID"`
}

type APIResponse struct {
	Id int `json:"id"`
	//Network       string `json:"network" bson:"network"`
	Plan_Name     string `json:"plan_name"`
	Plan_network  string `json:"plan_network"`
	Plan_amount   string `json:"plan_amount"`
	Mobile_number string `json:"mobile_number"`
	Ident         string `json:"ident"`
	Status        string `json:"Status"`
}

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

type SmileInfo struct {
	Network      string `json:"network"`
	Email        string `json:"email"`
	Phone_Number string `json:"phone_no"`
	AccountID    string `json:"accountID"` // Account ID, billlersCode
	Product      string `json:"product"`   //serviceID
	Product_plan string `json:"plan"`      // variation code
	RequestID    string `json:"request_id"`
}

type SmileAPIresponse struct {
	Code      string           `json:"code"`
	Content   smileContent     `json:"content"`
	Date      transaction_Date `json:"transaction_date"`
	RequestID string           `json:"requestId"`
	Response  string           `json:"response_description"`
}

type smileContent struct {
	Transcations smile_Transactions `json:"transactions"`
}

type smile_Transactions struct {
	Status        string  `json:"status"`
	Product_Desc  string  `json:"product_name"`
	Unit_Price    float64 `json:"unit_price"`
	Commission    float64 `json:"commission"`
	Email         string  `json:"email"`
	Phone         string  `json:"phone"`
	Amount        int     `json:"amount"`
	TransactionID string  `json:"transactionId"`
	Type          string  `json:"type"`
}

type transaction_Date struct {
	Date string `json:"date"`
}

type SmileResult struct {
	Network         string `json:"network" bson:"network"`
	ProductPlan     string `json:"plan" bson:"product"`
	Email           string `json:"email" bson:"email"`
	AccountID       string `json:"account_id" bson:"account_id"`
	Phone_Number    string `json:"phone_no" bson:"phone_no"`
	Name            string `json:"name" bson:"name"`
	Amount          int    `json:"amount" bson:"amount"`
	Product         string `json:"product" bson:"product"`
	Description     string `json:"description" bson:"description"`
	OrderID         int    `json:"order_id" bson:"order_id"`
	TranscationID   string `json:"transcation_id" bson:"transcation_id"`
	ReferenceNumber string `json:"Reference_number" bson:"reference_number"` // map transactionid from api to this.
	RequestID       string `json:"request_id" bson:"request_ID"`
}

type SpectranetInfo struct {
	Network      string `json:"network"`  // ServiceID
	Product      string `json:"product"`  //
	Plan         string `json:"plan"`     // variation code?
	Phone_Number string `json:"phone_no"` // billersCode &
	Name         string `json:"name"`
	No_of_Pins   string `json:"no_of_pins"` // quantity
	Amount       int    `json:"amount"`
	RequestID    string `json:"request_id"`
}

type SpectranetApiResponse struct {
	Code      string               `json:"code"`
	Content   spectranetContent    `json:"content"`
	Date      specTransaction_Date `json:"transaction_date"`
	RequestID string               `json:"requestId"`
	Response  string               `json:"response_description"`
	Cards     []card               `json:"cards"`
}

type spectranetContent struct {
	Transcations spec_Content `json:"transactions"`
}

type spec_Content struct {
	Status        string `json:"status"`
	Product_Desc  string `json:"product_name"`
	Phone_Number  string `json:"unique_element"`
	Unit_Price    int    `json:"unit_price"`
	Commission    int    `json:"commission"`
	Email         string `json:"email"`
	Phone         string `json:"phone"`
	Amount        int    `json:"amount"`
	Quantity      int    `json:"quantity"`
	TransactionID string `json:"transactionId"`
	Type          string `json:"type"`
}

type specTransaction_Date struct {
	Date string `json:"date"`
}

type card struct {
	SerialNumber string `json:"serialNumber"`
	Pin          string `json:"pin"`
	ExpiresOn    string `json:"expiresOn"`
	Value        int    `json:"value"`
}

type SpectranetResult struct {
	Network         string `json:"network" bson:"network"`
	Product         string `json:"product" bson:"product"`
	Plan            string `json:"plan" bson:"plan"`
	Email           string `json:"email" bson:"email"`
	Phone_Number    string `json:"phone_no" bson:"phone"`
	Name            string `json:"name" bson:"name"`
	No_of_Pins      int    `json:"no_of_pins" bson:"no_of_pins"`
	Amount          int    `json:"amount" bson:"amount"`
	ProductDesc     string `json:"product_desc" bson:"product_desc"`
	Description     string `json:"description" bson:"description"`
	OrderID         int    `json:"order_id" bson:"order_id"`
	TranscationID   string `json:"transcation_id" bson:"transaction_id"`
	ReferenceNumber string `json:"reference_number" bson:"reference_number"`
	RequestID       string `json:"request_id" bson:"request_ID"`
}
