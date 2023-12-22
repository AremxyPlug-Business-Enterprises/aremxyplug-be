package deposit

import "time"

type paymentResponse struct {
	Data []paymentData `json:"data"`
}

type paymentData struct {
	ID               string               `json:"id"`
	Type             string               `json:"type"`
	Attributes       paymentAttributes    `json:"attributes"`
	Relationships    paymentRelationships `json:"relationships"`
	Reference        string               `json:"reference"`
	PaymentReference string               `json:"paymentReference"`
	Currency         string               `json:"currency"`
	Amount           float64              `json:"amount"`
	Fee              int                  `json:"fee"`
	CreatedAt        time.Time            `json:"createdAt"`
	PaidAt           time.Time            `json:"paidAt"`
	Narration        string               `json:"narration"`
}

type paymentAttributes struct {
	CounterParty counterParty `json:"counterParty"`
}

type counterParty struct {
	ID            string    `json:"id"`
	Type          string    `json:"type"`
	CreatedAt     time.Time `json:"createdAt"`
	AccountNumber string    `json:"accountNumber"`
	AccountName   string    `json:"accountName"`
	Bank          bank      `json:"bank"`
}

type bank struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	CbnCode string `json:"cbnCode"`
	NipCode string `json:"nipCode"`
}

type paymentRelationships struct {
	SettlementAccount relationship  `json:"settlementAccount"`
	SubAccount        relationship  `json:"subAccount"`
	Customer          relationship  `json:"customer"`
	VirtualNuban      relationship  `json:"virtualNuban"`
	VirtualNubans     virtualNubans `json:"virtualNubans"`
	SubAccounts       subAccounts   `json:"subAccounts"`
	Settlements       settlements   `json:"settlements"`
}

type relationship struct {
	Data relationshipData `json:"data"`
}

type relationshipData struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

type virtualNubans struct {
	Data []relationshipData `json:"data"`
}

type subAccounts struct {
	Data []relationshipData `json:"data"`
}

type settlements struct {
	Data []relationshipData `json:"data"`
}
