package mongo

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aremxyplug-be/db/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	bankTransColl = "bank-transactions"
	balColl       = "balance"
	bankColl      = "bank"
	virtualColl   = "virtualAccount"
	counterColl   = "counterParty"
	deptColl      = "deposit"
)

var (
	ErrDepositIDExist = errors.New("deposit_id already exists")
)

func (m *mongoStore) deptColl() (*mongo.Collection, error) {
	col := m.mongoClient.Database(m.databaseName).Collection("deposit_IDs")
	ctx := context.Background()
	indexModel := mongo.IndexModel{
		Keys:    bson.D{primitive.E{Key: "ID", Value: 1}},
		Options: options.Index().SetUnique(true),
	}

	_, err := col.Indexes().CreateOne(ctx, indexModel)
	if err != nil {
		return nil, err
	}

	return col, nil
}

func (m *mongoStore) SaveBankList(banklist models.BankDetails) error {
	err := m.saveToDB(bankColl, banklist)
	return err
}

func (m *mongoStore) GetBankDetail(name string) (models.BankDetails, error) {
	ctx := context.Background()
	bankDetail := models.BankDetails{}

	bankName := strings.ToUpper(name)
	filter := bson.D{primitive.E{Key: "name", Value: bankName}}
	res := m.col(bankColl).FindOne(ctx, filter)

	err := res.Decode(&bankDetail)
	if err != nil {
		return models.BankDetails{}, err
	}

	return bankDetail, nil
}

func (m *mongoStore) SaveVirtualAccount(account models.AccountDetails) error {
	err := m.saveToDB(virtualColl, account)
	return err
}

func (m *mongoStore) GetVirtualNuban(name string) (models.AccountDetails, error) {
	ctx := context.Background()
	account_name := fmt.Sprintf("ANC(AREMXYPLUG/%s)", name)
	fmt.Println(account_name)
	filter := bson.D{primitive.E{Key: "account_name", Value: account_name}}

	/*
		filter = bson.D{primitive.E{Key: "user_id", Value: id}}
	*/

	acc_details := models.AccountDetails{}

	resp := m.col(virtualColl).FindOne(ctx, filter)
	if err := resp.Decode(&acc_details); err != nil {
		if err == mongo.ErrNoDocuments {
			return models.AccountDetails{}, nil
		}

		return models.AccountDetails{}, err
	}

	return acc_details, nil
}

func (m *mongoStore) SaveCounterParty(counterparty interface{}) error {
	err := m.saveToDB(counterColl, counterparty)
	return err
}

func (m *mongoStore) SaveTransfer(transfer models.TransferResponse) error {
	err := m.saveToDB(bankTransColl, transfer)
	return err
}

func (m *mongoStore) GetCounterParty(accountNumber, bankname string) (models.CounterParty, error) {
	ctx := context.Background()
	counterparty := models.CounterParty{}
	bankName := strings.ToUpper(bankname)

	// the filter should be using aggregate  search function since the fields that are to be acccessed are not on the top level.
	filter := bson.D{primitive.E{Key: "accountnumber", Value: accountNumber}, primitive.E{Key: "bankname", Value: bankName}}
	res := m.col(counterColl).FindOne(ctx, filter)

	err := res.Decode(&counterparty)
	if err != nil {
		return models.CounterParty{}, err
	}

	return counterparty, nil
}

func (m *mongoStore) GetTransferDetails(id string) (models.TransferResponse, error) {
	resp := m.getRecord(id, bankTransColl)
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

	findResult, err := m.getAllRecords(bankTransColl, user)
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
	resp := m.getRecord(id, bankTransColl)
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

	findResult, err := m.getAllRecords(bankTransColl, user)
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
	cur, err := m.getAllRecords(bankTransColl, user)
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
	err := m.saveToDB(bankTransColl, detail)
	return err
}

func (m *mongoStore) SaveDepositID(detail interface{}) error {
	ctx := context.Background()

	col, err := m.deptColl()
	if err != nil {
		return err
	}

	_, err = col.InsertOne(ctx, detail)
	if err != nil {
		if writeException, ok := err.(mongo.WriteException); ok {
			for _, writeError := range writeException.WriteErrors {
				var detailedError bson.Raw
				err := bson.Unmarshal([]byte(writeError.Error()), &detailedError)
				if err == nil {
					errMsg := detailedError.Lookup("errmsg").StringValue()
					fmt.Printf("Error: %s\n", errMsg)
				}
			}
			return ErrDepositIDExist
		} else {
			return err
		}
	}

	return nil
}

func (m *mongoStore) GetDepositID(virtualNuban string) (result interface{}, err error) {
	id_Result := m.getRecord(deptColl, virtualNuban)

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

	filter := bson.D{primitive.E{Key: "virtualnuban", Value: virtualNuban}}

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

	filter := bson.D{primitive.E{Key: "virtualnuban", Value: virtualNuban}}

	result := m.col(balColl).FindOne(ctx, filter)

	var resp models.Balance
	err := result.Decode(&resp)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// Collection or document not found, insert the new balance
			_, err := m.col(balColl).InsertOne(ctx, balance)
			if err != nil {
				return err
			}
			return nil
		}
		// Handle other errors
		return err
	}

	// Document found, update the existing balance
	updateFilter := bson.D{{Key: "$set", Value: bson.D{primitive.E{Key: "balance", Value: balance.Balance}}}}
	_, err = m.col(balColl).UpdateOne(ctx, filter, updateFilter)
	if err != nil {
		return err
	}

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

// first create the collection for pin
// code to save pin to the database
func (m *mongoStore) SavePin(data models.UserPin) error {
	ctx := context.Background()
	coll := m.col("pin")
	userColl := m.col("user")

	_, err := coll.InsertOne(ctx, data)
	if err != nil {
		return err
	}

	filter := bson.M{"id": data.UserID, "is_verified": false}
	update := bson.M{
		"$set": bson.M{
			"is_verified": true,
		},
	}

	updateResult, err := userColl.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update user document: %w", err)
	}

	if updateResult.MatchedCount == 0 {
		return errors.New("failed to update user document")
	}

	return nil
}

// code to get the pin from the database
func (m *mongoStore) GetPin(userID string) (string, error) {
	ctx := context.Background()
	filter := bson.D{primitive.E{Key: "userid", Value: userID}}

	result := m.col("pin").FindOne(ctx, filter)
	var resp models.UserPin
	err := result.Decode(&resp)
	if err != nil {
		if err == mongo.ErrNoDocuments {

			return "", nil
		}

		return "", err
	}

	return resp.Pin, nil
}

// code to update the pin in the database
func (m *mongoStore) UpdatePin(data models.UserPin) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.D{primitive.E{Key: "userid", Value: data.UserID}}

	updateFilter := bson.D{{Key: "$set", Value: bson.D{primitive.E{Key: "pin", Value: data.Pin}}}}

	_, err := m.col("pin").UpdateOne(ctx, filter, updateFilter)
	if err != nil {
		return err
	}

	return nil
}
