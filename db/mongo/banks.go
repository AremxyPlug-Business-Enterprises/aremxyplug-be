package mongo

import (
	"context"
	"time"

	"github.com/aremxyplug-be/db/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

var (
	bankTransColl = "bank-transactions"
	balColl       = "balance"
	bankColl      = "bank"
	virtualColl   = "virtualAccount"
	counterColl   = "counterParty"
)

func (m *mongoStore) SaveBankList(banklist []models.BankDetails) error {
	err := m.saveTransaction(bankColl, banklist)
	return err
}

func (m *mongoStore) GetBankDetail(bankName string) (models.BankDetails, error) {
	ctx := context.Background()
	bankDetail := models.BankDetails{}

	filter := bson.D{primitive.E{Key: "name", Value: bankName}}
	res := m.col(bankColl).FindOne(ctx, filter)

	err := res.Decode(&bankDetail)
	if err != nil {
		return models.BankDetails{}, err
	}

	return bankDetail, nil
}

func (m *mongoStore) SaveVirtualAccount(account models.AccountDetails) error {
	err := m.saveTransaction(virtualColl, account)
	return err
}

func (m *mongoStore) GetVirtualNuban(name string) (string, error) {
	ctx := context.Background()
	filter := bson.D{primitive.E{Key: "account_name", Value: name}}

	acc_details := models.AccountDetails{}

	resp := m.col(virtualColl).FindOne(ctx, filter)
	if err := resp.Decode(&acc_details); err != nil {
		if err == mongo.ErrNoDocuments {
			return "", nil
		}

		return "", err
	}

	return acc_details.Account_Name, nil
}

func (m *mongoStore) SaveCounterParty(counterparty interface{}) error {
	err := m.saveTransaction(counterColl, counterparty)
	return err
}

func (m *mongoStore) SaveTransfer(transfer models.TransferResponse) error {
	err := m.saveTransaction(bankTransColl, transfer)
	return err
}

func (m *mongoStore) GetCounterParty(accountNumber, bankname string) (models.CounterParty, error) {
	ctx := context.Background()
	counterparty := models.CounterParty{}

	// the filter should be using aggregate  search function since the fields that are to be acccessed are not on the top level.
	filter := bson.D{primitive.E{Key: "account_number", Value: accountNumber}, primitive.E{Key: "bank_name", Value: bankname}}
	res := m.col(counterColl).FindOne(ctx, filter)

	err := res.Decode(&counterparty)
	if err != nil {
		return models.CounterParty{}, err
	}

	return counterparty, nil
}

func (m *mongoStore) GetTransferDetails(id string) (models.TransferResponse, error) {
	resp := m.getTransaction(id, bankTransColl)
	result := models.TransferResponse{}
	err := resp.Decode(&result)
	if err != nil {
		return models.TransferResponse{}, err
	}

	return result, nil
}

func (m *mongoStore) GetAllTransferHistory(user string) ([]models.TransferResponse, error) {
	ctx := context.Background()
	result := []models.TransferResponse{}

	findResult, err := m.getAllTransaction(bankTransColl, user)
	if err != nil {
		return nil, err
	}

	for findResult.Next(ctx) {
		resp := models.TransferResponse{}
		if err := findResult.Decode(&resp); err != nil {
			return nil, err
		}

		result = append(result, resp)
	}
	defer findResult.Close(ctx)

	return result, nil

}

func (m *mongoStore) GetDepositDetails(id string) (models.DepositResponse, error) {
	resp := m.getTransaction(id, bankTransColl)
	result := models.DepositResponse{}
	err := resp.Decode(&result)
	if err != nil {
		return models.DepositResponse{}, err
	}

	return result, nil
}

func (m *mongoStore) GetAllDepositHistory(user string) ([]models.DepositResponse, error) {
	ctx := context.Background()
	result := []models.DepositResponse{}

	findResult, err := m.getAllTransaction(bankTransColl, user)
	if err != nil {
		return nil, err
	}

	for findResult.Next(ctx) {
		resp := models.DepositResponse{}
		if err := findResult.Decode(&resp); err != nil {
			return nil, err
		}

		result = append(result, resp)
	}
	defer findResult.Close(ctx)

	return result, nil
}

func (m *mongoStore) GetAllBankTransactions(user string) ([]interface{}, error) {
	ctx := context.Background()
	cur, err := m.getAllTransaction(bankTransColl, user)
	if err != nil {
		return nil, err
	}

	var transactionHistory []interface{}

	for cur.Next(ctx) {

		var raw bson.Raw

		if err := cur.Decode(&raw); err != nil {
			return nil, err
		}

		if raw.Lookup("").Type == bson.TypeString {
			var deposit models.DepositResponse

			if err := bson.Unmarshal(raw, &deposit); err != nil {
				return nil, err
			}

			transactionHistory = append(transactionHistory, deposit)
		} else if raw.Lookup("").Type == bson.TypeString {
			var tranfer models.TransferResponse

			if err := bson.Unmarshal(raw, &tranfer); err != nil {
				return nil, err
			}

			transactionHistory = append(transactionHistory, tranfer)
		}
	}

	if err := cur.Err(); err != nil {
		return nil, err
	}

	return transactionHistory, nil

}

func (m *mongoStore) SaveDeposit(detail models.DepositResponse) error {
	err := m.saveTransaction(bankTransColl, detail)
	return err
}

func (m *mongoStore) SaveDepositID(detail interface{}) error {
	err := m.saveTransaction("", detail)
	return err
}

func (m *mongoStore) GetDepositID(virtualNuban string) (result interface{}, err error) {
	id_Result := m.getTransaction(bankTransColl, virtualNuban)

	// change this result to struct
	var resp interface{}

	err = id_Result.Decode(&resp)
	if err == mongo.ErrNoDocuments {
		return "", nil
	} else if err != nil {
		return "", err
	}

	return resp, nil
}

func (m *mongoStore) GetBalance(virtualNuban string) (balance float64, err error) {

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.D{primitive.E{Key: "virtualNuban", Value: virtualNuban}}

	result := m.col(balColl).FindOne(ctx, filter)

	var bal models.Balance
	e := result.Decode(&bal)
	if e == mongo.ErrNoDocuments {
		return 0, nil
	} else if e != nil {
		return 0, e
	}

	return bal.Balance, nil
}

func (m *mongoStore) SaveBalance(virtualNuban string, balance models.Balance) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.D{primitive.E{Key: "virtualNuban", Value: virtualNuban}}

	result := m.col(balColl).FindOne(ctx, filter)

	var resp models.Balance
	err := result.Decode(&resp)
	if err != nil {
		return err
	} else if err == mongo.ErrNoDocuments {
		_, err := m.col(balColl).InsertOne(ctx, balance)
		if err != nil {
			return err
		}
	}

	updateFilter := bson.D{{Key: "$set", Value: bson.D{primitive.E{Key: "balance", Value: balance.Balance}}}}

	_, err = m.col(balColl).UpdateOne(ctx, filter, updateFilter)
	if err != nil {
		return err
	}

	// to update the balance, first from the struct you'll need to use a filter updating the balance in the document

	// should be used to update the balance.
	// first check if there was a previous balance, if non then save

	return nil
}

func (m *mongoStore) UpdateBalance(virtualNuban string, balance float64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.D{primitive.E{Key: "virtualNuban", Value: virtualNuban}}

	updateFilter := bson.D{{Key: "$set", Value: bson.D{primitive.E{Key: "balance", Value: balance}}}}

	_, err := m.col(balColl).UpdateOne(ctx, filter, updateFilter)
	if err != nil {
		return err
	}

	return nil
}
