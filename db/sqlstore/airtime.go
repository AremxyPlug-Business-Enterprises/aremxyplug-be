package sqlstore

import "github.com/aremxyplug-be/db/models"

func (s *SqlStore) GetAirtimeProduct(network string) (models.AirtimeProduct, error) {
	query := `SELECT network, provider_discount_percent, customer_discount_percent FROM airtime WHERE network = ?`
	var ap models.AirtimeProduct
	err := s.db.QueryRow(query, network).Scan(&ap.Network, &ap.Provider_Discount, &ap.Customer_Discount)
	if err != nil {
		return models.AirtimeProduct{}, err
	}
	return ap, nil
}
