package transfer

type intiateTransfer struct {
	Data transferData `json:"data"`
}

type transferData struct {
	Attributes    transferDataAttributes `json:"attributes"`
	Relationships relationships          `json:"relationships"`
	Type          string                 `json:"type"`
}

type transferDataAttributes struct {
	Currency  string  `json:"currency"`
	Amount    float64 `json:"amount"`
	Reason    string  `json:"reason,omitempty"`
	Reference string  `json:"reference,omitempty"`
}

type account struct {
	Data data `json:"data"`
}

type relationships struct {
	DestinationAcc destination  `json:"destinationAccount"`
	Account        account      `json:"account"`
	CounterParty   counterParty `json:"counterParty"`
}

type destination struct {
	Data struct {
		Type string `json:"type"`
	} `json:"data"`
}

type counterParty struct {
	Data data `json:"data"`
}

type data struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

type transferResult struct {
	Data transferResultData `json:"data"`
}

type transferResultData struct {
	Type       string                   `json:"type"`
	Attributes transferResultAttributes `json:"attributes"`
}

type transferResultAttributes struct {
	Reason        string  `json:"reason"`
	FailureReason string  `json:"failureReason"`
	Ammount       float64 `json:"ammount"`
	Status        string  `json:"status"`
}

type counterPartyAPIResponse struct {
	Data counterpartyResponseData `json:"data"`
}

type counterpartyResponseData struct {
	Type       string                         `json:"type"`
	ID         string                         `json:"id"`
	Attributes counterpartyResponseAttributes `json:"attributes"`
}

type counterpartyResponseAttributes struct {
	AccountName   string `json:"accountName"`
	AccountNumber string `json:"accountNumber"`
	Status        string `json:"status"`
	Bank          bank   `json:"bank"`
}

type bank struct {
	Name    string `json:"name"`
	NipCode string `json:"nipCode"`
}

type verifyAccountResponse struct {
	Data verifyAccountData `json:"data"`
}

type verifyAccountData struct {
	Attributes verifyAccountAttributes `json:"attributes"`
}

type verifyAccountAttributes struct {
	Bank          bank   `json:"bank"`
	AccountName   string `json:"accountName"`
	AccountNumber string `json:"accountNumber"`
}

type counterPartyPayload struct {
	Data counterpartyData `json:"data"`
}

type counterpartyData struct {
	Type          string                   `json:"type"`
	Attributes    counterpartyAttributes   `json:"attributes"`
	Relationships counterpartyRelationship `json:"relationships"`
}

type counterpartyAttributes struct {
	VerifyName    bool   `json:"verifyName"`
	AccountName   string `json:"accountName"`
	AccountNumber string `json:"accountNumber"`
	BankCode      string `json:"bankCode"`
}

type counterpartyRelationship struct {
	Bank struct {
		Data accountData `json:"data"`
	} `json:"bank"`
}

type accountData struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

type bankLists struct {
	BanksData []bankData `json:"data"`
}

type bankAttributes struct {
	NIPCode string `json:"nipCode"`
	Name    string `json:"name"`
}

type bankData struct {
	ID         string         `json:"id"`
	Type       string         `json:"type"`
	Atrributes bankAttributes `json:"attributes"`
}

type AremxyPlugTransfer struct {
	UserID   string
	Username string  `json:"username,omitempty"`
	Email    string  `json:"email,omitempty"`
	Amount   float64 `json:"amount"`
	Phone    string  `json:"phone"`
	Name     string  `json:"name"`
	Reason   string  `json:"reason,omitempty"`
	FullName string
}

type TransferInfo struct {
	UserID         string
	Bank_name      string  `json:"bank_name"`
	Account_Number string  `json:"account_number"`
	Account_Name   string  `json:"account_name"`
	Amount         float64 `json:"amount"`
	Reason         string  `json:"message"`
	FullName       string
	Source         string
	Email          string
	Phone          string
	Username       string
	RecipientName  string
}
