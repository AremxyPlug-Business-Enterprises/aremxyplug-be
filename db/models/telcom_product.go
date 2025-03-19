package models

type Product struct {
	Product_ID int
	Network_ID int
	Plan_Type  string
}

type Plan struct {
	PlanID    int
	ProductID int
	Amount    float64
	Validity  string
	Size      string
}
