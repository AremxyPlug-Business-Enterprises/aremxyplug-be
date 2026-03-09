package sqlstore

import "github.com/aremxyplug-be/db/models"

func (s *SqlStore) GetAirtimeProduct(network string) (models.AirtimeProduct, error) {
	query := `SELECT network, provider_discount_percent, customer_discount_percent, profit_margin FROM airtime WHERE network = $1`
	var ap models.AirtimeProduct
	err := s.db.QueryRow(query, network).Scan(&ap.Network, &ap.Provider_Discount, &ap.Customer_Discount, &ap.Profit_Margin)
	if err != nil {
		return models.AirtimeProduct{}, err
	}
	return ap, nil
}
