package models

type TVSub struct {
	ID          int     `db:"plan_id"`
	Package     string  `db:"package"`
	PackageName string  `db:"package_name"`
	Amount      float64 `db:"amount"`
}

type TVSubUpdate struct {
	Package     *string
	PackageName *string
	Amount      *float64
}
