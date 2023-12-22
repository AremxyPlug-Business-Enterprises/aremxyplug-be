package transactions

import (
	"github.com/aremxyplug-be/db"
	"github.com/aremxyplug-be/db/models"
)

// all functions here are to call from the database

type Transaction struct {
	store db.DataStore
}

func NewTransaction(store db.DataStore) *Transaction {
	return &Transaction{
		store: store,
	}
}

func (t *Transaction) GetTransferDetails(id string) (models.TransferResponse, error) {
	result, err := t.store.GetTransferDetails(id)
	if err != nil {
		// log error
		return models.TransferResponse{}, err
	}

	return result, nil
}

func (t *Transaction) GetTransferHistory(user string) ([]models.TransferResponse, error) {
	result, err := t.store.GetAllTransferHistory(user)
	if err != nil {
		// log error
		return nil, err
	}

	return result, nil
}

func (t *Transaction) GetAllTransactionHistory() ([]interface{}, error) {
	result, err := t.store.GetAllBankTransactions("")
	if err != nil {
		// log error
		return nil, err
	}

	return result, nil

}

func (t *Transaction) GetDepositHistory(user string) ([]models.DepositResponse, error) {
	result, err := t.store.GetAllDepositHistory(user)
	if err != nil {
		// log error
		return nil, err
	}

	return result, nil
}

func (t *Transaction) GetDepositDetails(id string) (models.DepositResponse, error) {
	result, err := t.store.GetDepositDetails(id)
	if err != nil {
		// log error
		return models.DepositResponse{}, err
	}

	return result, nil
}

// should be called at any point where the user get their balance
func (t *Transaction) GetBalance(virtualNuban string) (float64, error) {
	bal, err := t.store.GetBalance(virtualNuban)
	if err != nil {
		return 0, err
	}
	return bal, nil
}

// To be used after making payment
func (t *Transaction) UpdateBalance(virtualNuban string, amount float64) error {
	err := t.store.UpdateBalance(virtualNuban, amount)
	if err != nil {
		return err
	}

	return nil
}
