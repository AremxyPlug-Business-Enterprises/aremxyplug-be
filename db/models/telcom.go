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

type EduInfo struct {
	Exam_Type    string `json:"exam_type"`
	Phone_Number string `json:"phone_no"`
	Amount       string `json:"amount"`
	Email        string `json:"email"`
	Quantity     int    `json:"quantity"`
	Wallet_Type  string `json:"wallet_type"`
}

type EduApiResponse struct {
	Message          string  `json:"message"`
	Amount           float64 `json:"amount"`
	Date             string  `json:"transaction_date"`
	Status           string  `json:"status"`
	Reference        string  `json:"reference_no"`
	Pin              Pins
	Success_Response string `json:"success"`
}

type Pins struct {
	Pin1 string `json:"pin1"`
	Pin2 string `json:"pin2"`
	Pin3 string `json:"pin3"`
	Pin4 string `json:"pin4"`
	Pin5 string `json:"pin5"`
}

type EduResponse struct {
	OrderID         int      `json:"order_id" bson:"order_id"`
	Email           string   `json:"email" bson:"email"`
	Phone           string   `json:"phone_no" bson:"phone_no"`
	TransactionID   string   `json:"transaction_id"`
	ReferenceNumber string   `json:"reference_no" bson:"reference_no"`
	Product         string   `json:"product" bson:"product"`
	Amount          float64  `json:"amount" bson:"amount"`
	Exam_Type       string   `json:"exam_type" bson:"exam_type"`
	Description     string   `json:"description" bson:"description"`
	Status          string   `json:"status" bson:"status"`
	Pin_Generated   []string `json:"pins_generated" bson:"pins_generated"`
	CreatedAt       string   `json:"created_at" bson:"created_at"`
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
	Content   SmileContent     `json:"content"`
	Date      Transaction_Date `json:"transaction_date"`
	RequestID string           `json:"requestId"`
	Response  string           `json:"response_description"`
}

type SmileContent struct {
	Transcations Smile_Transactions `json:"transactions"`
}

type Smile_Transactions struct {
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

type Transaction_Date struct {
	Date string `json:"date"`
}

type SmileResult struct {
	Network         string `json:"network"`
	ProductPlan     string `json:"plan"`
	Email           string `json:"email"`
	AccountID       string `json:"account_id"`
	Phone_Number    string `json:"phone_no"`
	Name            string `json:"name"`
	Amount          int    `json:"amount"`
	Product         string `json:"product"`
	Description     string `json:"description"`
	OrderID         int    `json:"order_id"`
	TranscationID   string `json:"transcation_id"`
	ReferenceNumber string `json:"Reference_number"` // map transactionid from api to this.
	RequestID       string `json:"request_id"`
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

type SpectranetApi struct {
	Code      string               `json:"code"`
	Content   SpectranetContent    `json:"content"`
	Date      SpecTransaction_Date `json:"transaction_date"`
	RequestID string               `json:"requestId"`
	Response  string               `json:"response_description"`
	Cards     []Card               `json:"cards"`
}

type SpectranetContent struct {
	Transcations Spec_Content `json:"transactions"`
}

type Spec_Content struct {
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

type SpecTransaction_Date struct {
	Date string `json:"date"`
}

type Card struct {
	SerialNumber string `json:"serialNumber"`
	Pin          string `json:"pin"`
	ExpiresOn    string `json:"expiresOn"`
	Value        int    `json:"value"`
}

type SpectranetResult struct {
	Network         string `json:"network"`
	Product         string `json:"product"`
	Plan            string `json:"plan"`
	Email           string `json:"email"`
	Phone_Number    string `json:"phone_no"`
	Name            string `json:"name"`
	No_of_Pins      int    `json:"no_of_pins"`
	Amount          int    `json:"amount"`
	ProductDesc     string `json:"product_desc"`
	Description     string `json:"description"`
	OrderID         int    `json:"order_id"`
	TranscationID   string `json:"transcation_id"`
	ReferenceNumber string `json:"reference_number"`
	RequestID       string `json:"request_id"`
}
