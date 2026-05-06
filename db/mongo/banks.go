package mongo

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/aremxyplug-be/db/models"
	"github.com/shopspring/decimal"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	depositColl  = "deposit-transactions"
	transferColl = "transfer"
	balColl      = "balance"
	bankColl     = "bank"
	virtualColl  = "virtualAccount"
	counterColl  = "counterParty"
	deptColl     = "deposit_IDs"
	tasksColl    = "tasks"
)

var (
	ErrDepositIDExist = errors.New("deposit_id already exists")
)

func (m *mongoStore) SaveBankList(ctx context.Context, banklist models.BankDetails) error {
	ctx = m.ensureCtx(ctx)
	err := m.saveToDB(ctx, bankColl, banklist)
	return err
}

func (m *mongoStore) GetAllBanks(ctx context.Context) ([]models.BankDetails, error) {
	ctx = m.ensureCtx(ctx)
	bankColl := m.col(bankColl)
	cursor, err := bankColl.Find(ctx, bson.D{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var banks []models.BankDetails
	for cursor.Next(ctx) {
		var bank models.BankDetails
		if err := cursor.Decode(&bank); err != nil {
			return nil, err
		}
		banks = append(banks, bank)
	}
	return banks, nil
}

func (m *mongoStore) UpsertBankByNIPCode(ctx context.Context, bank models.BankDetails) error {
	ctx = m.ensureCtx(ctx)
	filter := bson.M{"nip_code": bank.NIPCode}
	update := bson.M{"$set": bson.M{"name": bank.Name}}
	bankColl := m.col(bankColl)
	opts := options.Update().SetUpsert(true)
	_, err := bankColl.UpdateOne(ctx, filter, update, opts)
	return err
}

func (m *mongoStore) DeleteBankByNIPCode(ctx context.Context, nipCode string) error {
	ctx = m.ensureCtx(ctx)
	filter := bson.D{primitive.E{Key: "nip_code", Value: nipCode}}
	bankColl := m.col(bankColl)

	_, err := bankColl.DeleteOne(ctx, filter)
	return err
}

func (m *mongoStore) GetBankByNIPCode(ctx context.Context, nipCode string) (*models.BankDetails, error) {
	ctx = m.ensureCtx(ctx)
	filter := bson.D{primitive.E{Key: "nip_code", Value: nipCode}}
	var bank models.BankDetails
	bankColl := m.col(bankColl)
	if err := bankColl.FindOne(ctx, filter).Decode(&bank); err != nil {
		return nil, err
	}
	return &bank, nil
}

func (m *mongoStore) UpdateBank(ctx context.Context, bank models.BankDetails) error {
	ctx = m.ensureCtx(ctx)
	filter := bson.D{primitive.E{Key: "nip_code", Value: bank.NIPCode}}
	update := bson.D{primitive.E{Key: "$set", Value: bson.D{primitive.E{Key: "name", Value: bank.Name}}}}
	bankColl := m.col(bankColl)
	_, err := bankColl.UpdateOne(ctx, filter, update)
	return err
}

func (m *mongoStore) GetBankDetail(ctx context.Context, name string) (models.BankDetails, error) {
	ctx = m.ensureCtx(ctx)
	bankDetail := models.BankDetails{}

	//bankName := strings.ToUpper(name)
	filter := bson.D{primitive.E{Key: "name", Value: name}}
	res := m.col(bankColl).FindOne(ctx, filter)

	err := res.Decode(&bankDetail)
	if err != nil {
		return models.BankDetails{}, err
	}

	return bankDetail, nil
}

func (m *mongoStore) SaveVirtualAccount(ctx context.Context, account models.AccountDetails) error {
	ctx = m.ensureCtx(ctx)
	if strings.TrimSpace(account.Status) == "" {
		account.Status = "active"
	}
	// Start session
	session, err := m.mongoClient.StartSession()
	if err != nil {
		return fmt.Errorf("failed to start session: %w", err)
	}
	defer session.EndSession(ctx)

	// Transaction operation
	_, err = session.WithTransaction(ctx, func(ctx mongo.SessionContext) (interface{}, error) {

		if err := m.saveToDB(ctx, virtualColl, account); err != nil {
			return nil, fmt.Errorf("failed to save account: %w", err)
		}

		userColl := m.col("user")
		filter := bson.M{
			"id":                account.User_ID,
			"has_virtual_nuban": false,
		}

		update := bson.M{"$set": bson.M{"has_virtual_nuban": true}}

		result, err := userColl.UpdateOne(ctx, filter, update)
		if err != nil {
			return nil, fmt.Errorf("failed to update user: %w", err)
		}

		if result.MatchedCount == 0 {
			return nil, errors.New("user not found or already has virtual account")
		}

		return nil, nil
	})

	return err
}

func (m *mongoStore) GetVirtualNuban(ctx context.Context, id string) (models.AccountDetails, error) {
	ctx = m.ensureCtx(ctx)
	filter := bson.M{"user_id": id}

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

func (m *mongoStore) SaveCounterParty(ctx context.Context, counterparty interface{}) error {
	ctx = m.ensureCtx(ctx)
	err := m.saveToDB(ctx, counterColl, counterparty)
	return err
}

func (m *mongoStore) SaveTransfer(ctx context.Context, transfer models.TransferResponse) error {
	ctx = m.ensureCtx(ctx)
	err := m.saveToDB(ctx, transferColl, transfer)
	return err
}

func (m *mongoStore) GetCounterParty(ctx context.Context, accountNumber, bankname string) (models.CounterParty, error) {
	ctx = m.ensureCtx(ctx)
	counterparty := models.CounterParty{}
	//bankName := strings.ToUpper(bankname)

	// the filter should be using aggregate  search function since the fields that are to be acccessed are not on the top level.
	filter := bson.D{primitive.E{Key: "account_number", Value: accountNumber}, primitive.E{Key: "bank_name", Value: bankname}}
	res := m.col(counterColl).FindOne(ctx, filter)

	err := res.Decode(&counterparty)
	if err != nil {
		return models.CounterParty{}, err
	}

	return counterparty, nil
}

func (m *mongoStore) GetTransferDetails(ctx context.Context, id string) (models.TransferResponse, error) {
	ctx = m.ensureCtx(ctx)
	resp := m.getRecord(ctx, id, transferColl)
	result := models.TransferResponse{}
	err := resp.Decode(&result)
	if err != nil {
		return models.TransferResponse{}, err
	}

	return result, nil
}

func (m *mongoStore) GetAllTransferHistory(ctx context.Context, userID string) ([]models.TransferResponse, error) {
	ctx = m.ensureCtx(ctx)
	result := []models.TransferResponse{}

	findResult, err := m.getAllRecords(ctx, transferColl, userID)
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

func (s *mongoStore) GetDepositDetails(ctx context.Context, id string) (any, error) {
	ctx = s.ensureCtx(ctx)
	// First decode only transaction_product so we can choose the proper response shape.
	var probe struct {
		TransactionProduct string `bson:"transaction_product"`
	}
	result := s.getRecord(ctx, id, depositColl)
	err := result.Decode(&probe)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return models.DepositResponse{}, nil
		}
		return models.DepositResponse{}, err
	}

	if strings.EqualFold(probe.TransactionProduct, "Internal Deposit") ||
		strings.EqualFold(probe.TransactionProduct, "System Top-Up") || strings.EqualFold(probe.TransactionProduct, "System Debit") {
		var internalRes models.InternalDepositResponse
		internalResult := s.getRecord(ctx, id, depositColl)
		if err := internalResult.Decode(&internalRes); err != nil {
			if err == mongo.ErrNoDocuments {
				return models.InternalDepositResponse{}, nil
			}
			return models.InternalDepositResponse{}, err
		}
		return internalRes, nil
	}

	var res models.DepositResponse
	defaultResult := s.getRecord(ctx, id, depositColl)
	if err := defaultResult.Decode(&res); err != nil {
		if err == mongo.ErrNoDocuments {
			return models.DepositResponse{}, nil
		}
		return models.DepositResponse{}, err
	}

	return res, nil
}

func (m *mongoStore) GetAllDepositHistory(ctx context.Context, userID string) ([]models.DepositResponse, error) {
	ctx = m.ensureCtx(ctx)
	result := []models.DepositResponse{}

	findResult, err := m.getAllRecords(ctx, depositColl, userID)
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

func (m *mongoStore) GetAllBankTransactions(ctx context.Context, userID string) ([]interface{}, error) {
	ctx = m.ensureCtx(ctx)
	var transactions []interface{}

	// Fetch deposits
	depositCursor, err := m.col(depositColl).Find(ctx, bson.D{{Key: "user_id", Value: userID}})
	if err != nil {
		return nil, fmt.Errorf("failed to query deposit collection: %w", err)
	}
	defer depositCursor.Close(ctx)

	for depositCursor.Next(ctx) {
		var deposit models.DepositResponse
		if err := depositCursor.Decode(&deposit); err != nil {
			return nil, fmt.Errorf("failed to decode deposit record: %w", err)
		}
		transactions = append(transactions, deposit)
	}

	// Fetch transfers
	transferCursor, err := m.col(transferColl).Find(ctx, bson.D{{Key: "user_id", Value: userID}})
	if err != nil {
		return nil, fmt.Errorf("failed to query transfer collection: %w", err)
	}
	defer transferCursor.Close(ctx)

	for transferCursor.Next(ctx) {
		var transfer models.TransferResponse
		if err := transferCursor.Decode(&transfer); err != nil {
			return nil, fmt.Errorf("failed to decode transfer record: %w", err)
		}
		transactions = append(transactions, transfer)
	}

	// Check cursor errors
	if err := depositCursor.Err(); err != nil {
		return nil, fmt.Errorf("deposit cursor error: %w", err)
	}
	if err := transferCursor.Err(); err != nil {
		return nil, fmt.Errorf("transfer cursor error: %w", err)
	}

	// Sort transactions by time (newest first)
	sort.Slice(transactions, func(i, j int) bool {
		return getTransactionTime(transactions[i]).After(getTransactionTime(transactions[j]))
	})

	return transactions, nil
}

func getTransactionTime(record interface{}) time.Time {
	switch t := record.(type) {
	case models.DepositResponse:
		return t.CreatedAt
	case models.TransferResponse:
		return t.CreatedAt
	default:
		return time.Time{}
	}
}
func (m *mongoStore) SaveDeposit(ctx context.Context, detail any) error {
	ctx = m.ensureCtx(ctx)
	err := m.saveToDB(ctx, depositColl, detail)
	return err
}

func (m *mongoStore) SaveDepositID(ctx context.Context, detail interface{}) error {
	ctx = m.ensureCtx(ctx)

	col := m.col(deptColl)

	_, err := col.InsertOne(ctx, detail)
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

func (m *mongoStore) GetDepositID(ctx context.Context, virtualNuban string) (result interface{}, err error) {
	ctx = m.ensureCtx(ctx)
	id_Result := m.getRecord(ctx, deptColl, virtualNuban)

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

func (m *mongoStore) CreateInitialBalance(ctx context.Context, userID, virtualNuban string) error {
	ctx = m.ensureCtx(ctx)

	balance, err := primitive.ParseDecimal128("0.00")
	if err != nil {
		m.logger.Error(err.Error())
		return err
	}

	initialBalance := models.Balance{
		VirtualNuban: virtualNuban,
		UserID:       userID,
		Balance:      balance,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	_, err = m.col(balColl).InsertOne(ctx, initialBalance)
	return err
}

func (m *mongoStore) GetBalance(ctx context.Context, userID string) (balance decimal.Decimal, err error) {
	ctx = m.ensureCtx(ctx)

	filter := bson.D{primitive.E{Key: "user_id", Value: userID}}

	result := m.col(balColl).FindOne(ctx, filter)

	var bal models.Balance
	e := result.Decode(&bal)
	if e == mongo.ErrNoDocuments {
		return decimal.Decimal{}, nil
	} else if e != nil {
		return decimal.Decimal{}, e
	}

	retrievedBalance, err := decimal.NewFromString(bal.Balance.String())
	if err != nil {
		return decimal.Decimal{}, err
	}

	return retrievedBalance, nil
}

func (m *mongoStore) GetBalanceDetails(ctx context.Context, id string) (models.Balance, error) {
	ctx = m.ensureCtx(ctx)
	filter := bson.D{primitive.E{Key: "user_id", Value: id}}

	result := m.col(balColl).FindOne(ctx, filter)
	var resp models.Balance
	err := result.Decode(&resp)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return models.Balance{}, nil
		}
		return models.Balance{}, err
	}

	return resp, nil
}

func (m *mongoStore) SaveBalance(ctx context.Context, userID string, balance models.Balance) error {
	ctx = m.ensureCtx(ctx)

	filter := bson.D{primitive.E{Key: "user_id", Value: userID}}

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

func (m *mongoStore) UpdateBalance(ctx context.Context, userID string, balance decimal.Decimal) error {
	ctx = m.ensureCtx(ctx)

	filter := bson.D{primitive.E{Key: "user_id", Value: userID}}

	bal, err := primitive.ParseDecimal128(balance.String())
	if err != nil {
		m.logger.Error(err.Error())
		return err
	}

	updateFilter := bson.D{{Key: "$set", Value: bson.D{
		{Key: "balance", Value: bal},
		{Key: "updated_at", Value: time.Now().UTC()},
	}}}

	_, err = m.col(balColl).UpdateOne(ctx, filter, updateFilter)
	if err != nil {
		return err
	}

	return nil
}

// first create the collection for pin
// code to save pin to the database
func (m *mongoStore) SavePin(ctx context.Context, data models.UserPin) error {
	ctx = m.ensureCtx(ctx)
	coll := m.col("pin")
	userColl := m.col("user")

	_, err := coll.InsertOne(ctx, data)
	if err != nil {
		return err
	}

	filter := bson.M{"id": data.UserID, "has_Pin": false}
	update := bson.M{
		"$set": bson.M{
			"has_Pin": true,
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
func (m *mongoStore) GetPin(ctx context.Context, userID string) (string, error) {
	ctx = m.ensureCtx(ctx)
	filter := bson.D{primitive.E{Key: "user_id", Value: userID}}

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

func (m *mongoStore) UpdatePin(ctx context.Context, data models.UserPin) error {
	ctx = m.ensureCtx(ctx)

	filter := bson.D{primitive.E{Key: "user_id", Value: data.UserID}}

	// need to update update the updated_at field as well

	updateFilter := bson.D{{Key: "$set", Value: bson.D{primitive.E{Key: "pin", Value: data.Pin}, {Key: "updated_at", Value: time.Now().UTC()}}}}

	_, err := m.col("pin").UpdateOne(ctx, filter, updateFilter)
	if err != nil {
		return err
	}

	return nil
}

func (m *mongoStore) SaveTransferRecipient(ctx context.Context, userID, username, email, phone, fullName string) error {
	ctx = m.ensureCtx(ctx)
	col := m.col("transfer_recipients")

	filter := bson.M{"user_id": userID}

	var existing models.TransferRecipient
	err := col.FindOne(ctx, filter).Decode(&existing)

	newRecipient := models.TransferRecipientDetails{
		Username: username,
		Email:    email,
		Phone:    phone,
		FullName: fullName,
	}

	if err != nil {
		if err == mongo.ErrNoDocuments {
			// New user, create document
			newDoc := models.TransferRecipient{
				UserID:    userID,
				Recipient: []models.TransferRecipientDetails{newRecipient},
				CreatedAt: time.Now(),
			}
			_, err := col.InsertOne(ctx, newDoc)
			if err != nil {
				return fmt.Errorf("failed to insert new recipient document: %w", err)
			}
			return nil
		}
		return fmt.Errorf("failed to check existing recipient document: %w", err)
	}

	// Check if recipient already exists for the user
	for _, r := range existing.Recipient {
		if r.Email == email {
			return fmt.Errorf("recipient already saved")
		}
	}

	// Add new recipient to slice
	update := bson.M{
		"$push": bson.M{
			"recipient": newRecipient,
		},
	}
	_, err = col.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update recipient list: %w", err)
	}

	return nil
}

var ErrNoRecipientFound = errors.New("no recipient found for user")

func (m *mongoStore) GetTransferRecipients(ctx context.Context, userID string) ([]models.TransferRecipientDetails, error) {
	ctx = m.ensureCtx(ctx)
	col := m.col("transfer_recipients")

	var existing models.TransferRecipient
	err := col.FindOne(ctx, bson.M{"user_id": userID}).Decode(&existing)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrNoRecipientFound
		}
		return nil, fmt.Errorf("failed to retrieve recipients: %w", err)
	}

	return existing.Recipient, nil
}

func (m *mongoStore) DeleteTransferRecipient(ctx context.Context, userID, email string) error {
	ctx = m.ensureCtx(ctx)
	col := m.col("transfer_recipients")

	filter := bson.M{"user_id": userID}
	update := bson.M{
		"$pull": bson.M{
			"recipient": bson.M{"email": email},
		},
	}

	result, err := col.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to delete recipient: %w", err)
	}
	if result.ModifiedCount == 0 {
		return fmt.Errorf("recipient not found or already deleted")
	}

	return nil
}
