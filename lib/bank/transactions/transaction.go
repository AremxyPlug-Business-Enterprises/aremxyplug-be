package transactions

import (
	"context"

	"github.com/aremxyplug-be/db"
	"github.com/aremxyplug-be/db/models"
	"github.com/shopspring/decimal"
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

func (t *Transaction) GetTransferDetails(ctx context.Context, id string) (models.TransferResponse, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	result, err := t.store.GetTransferDetails(ctx, id)
	if err != nil {
		// log error
		return models.TransferResponse{}, err
	}

	return result, nil
}

func (t *Transaction) GetTransferHistory(ctx context.Context, user string) ([]models.TransferResponse, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	result, err := t.store.GetAllTransferHistory(ctx, user)
	if err != nil {
		// log error
		return nil, err
	}

	return result, nil
}

func (t *Transaction) GetAllTransactionHistory(ctx context.Context) ([]interface{}, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	result, err := t.store.GetAllBankTransactions(ctx, "")
	if err != nil {
		// log error
		return nil, err
	}

	return result, nil

}

func (t *Transaction) GetDepositHistory(ctx context.Context, user string) ([]models.DepositResponse, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	result, err := t.store.GetAllDepositHistory(ctx, user)
	if err != nil {
		// log error
		return nil, err
	}

	return result, nil
}

func (t *Transaction) GetDepositDetails(ctx context.Context, id string) (any, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	result, err := t.store.GetDepositDetails(ctx, id)
	if err != nil {
		// log error
		return nil, err
	}

	return result, nil
}

// should be called at any point where the user get their balance
func (t *Transaction) GetBalance(ctx context.Context, userID string) (decimal.Decimal, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	bal, err := t.store.GetBalance(ctx, userID)
	if err != nil {
		return decimal.NewFromFloat(0), err
	}
	return bal, nil
}

// To be used after making payment
func (t *Transaction) UpdateBalance(ctx context.Context, userID string, amount float64) error {
	if ctx == nil {
		ctx = context.Background()
	}
	err := t.store.UpdateBalance(ctx, userID, decimal.NewFromFloatWithExponent(amount, -2))
	if err != nil {
		return err
	}

	return nil
}
