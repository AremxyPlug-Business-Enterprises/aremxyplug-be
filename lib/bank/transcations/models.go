package transactions

type depositApiResponse struct {
	Data depositApiResponseData `json:"data"`
}

type depositApiResponseData struct {
	ID           string `json:"id"`
	Type         string `json:"type"`
	Attributes   depositAttributes
	Reference    string        `json:"reference"`
	Amount       float64       `json:"amount"`
	Fee          float64       `json:"fee"`
	Relationship relationships `json:"relationships"`
}

type depositAttributes struct {
	CounterParty counterParty `json:"counterParty"`
}

type counterParty struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

type relationships struct {
	Settlement settlementAccount `json:"settlementAccount"`
}

type bank struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	NipCode string `json:"nipCode"`
}

type settlementAccount struct {
	Data data `json:"data"`
}

type data struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}
