package models

type TransferInfo struct {
	Bank_name      string `json:"bank_name"`
	Account_Number string `json:"account_number"`
	Account_Name   string `json:"account_name"`
	Amount         string `json:"amount"`
	Reason         string `json:"message"`
}

type TransferResponse struct {
	Bank_Name      string `json:"bank_name"`
	Account_Name   string `json:"account_name"`
	Account_No     string `json:"account_no"`
	Name           string `json:"name"`
	Product        string `json:"product"`
	Description    string `json:"description"`
	Reason         string `json:"message"`
	Order_ID       int    `json:"order_id"`
	Transaction_ID string `json:"transaction_id"`
	Session_ID     string `json:"session_id"`
}

type AccountDetails struct {
	Bank_Name    string `json:"bank_name"`
	Account_Name string `json:"account_name"`
	Account_No   string `json:"account_no"`
}

type CounterParty struct {
	ID            string `json:"id"`
	AccountName   string `json:"account_name"`
	AccountNumber string `json:"account_number"`
	BankName      string `json:"bank_name"`
	NIPCode       string `json:"nipCode"`
}
type BankDetails struct {
	Name    string `json:"name"`
	NIPCode string `json:"nipCode"`
}

type DepositResponse struct {
	Amount         string `json:"amount"`         // amount recieved
	WalletType     string `json:"walletType"`     // Nigerian NGN wallet
	Bank_Name      string `json:"bank_name"`      // sender's bank name
	Account_Name   string `json:"account_name"`   // sender's account name
	Account_No     string `json:"account_no"`     // sender's account number
	Product        string `json:"product"`        // *Virtual account
	Description    string `json:"description"`    // description based on the method of deposit
	Message        string `json:"message"`        // map to narration
	Order_ID       int    `json:"order_id"`       // orderID created
	Transaction_ID string `json:"transaction_id"` // transactionID created
	Session_ID     string `json:"session_id"`     // map to paymentReference
}

type Balance struct {
	VirtualNuban string  `json:"virtualNuban"`
	Balance      float64 `json:"balance"`
}
