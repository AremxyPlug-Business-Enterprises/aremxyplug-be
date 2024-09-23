package telcom

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
