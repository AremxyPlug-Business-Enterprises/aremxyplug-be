package models

type TVSub struct {
	ID            int      `db:"plan_id"`
	Package       string   `db:"package"`
	PackageName   string   `db:"package_name"`
	Amount        float64  `db:"amount"`
	Profit_Margin *float64 `db:"profit_margin"`
}

type TVSubUpdate struct {
	Package       *string
	PackageName   *string
	Amount        *float64
	Profit_Margin *float64
}
