package deposit

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/aremxyplug-be/db"
	"github.com/aremxyplug-be/db/models"
	"github.com/aremxyplug-be/db/mongo"
	"github.com/aremxyplug-be/db/sqlstore"
	"github.com/aremxyplug-be/lib/balance"
	"github.com/aremxyplug-be/lib/events"
	"github.com/aremxyplug-be/lib/randomgen"
	"github.com/shopspring/decimal"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

var (
	api    = os.Getenv("ANCHOR_API")
	apikey = os.Getenv("ANCHORAPI_PROD")
)

type Config struct {
	db        db.DataStore
	sqlStore  *sqlstore.SqlStore
	logger    *zap.Logger
	processor *events.Processor
}

type depositID struct {
	VirtualNuban string `json:"virtualNuban" bson:"virtualNuban"`
	ID           string `json:"id" bson:"ID"`
}

func NewDepositConfig(db db.DataStore, sqlStore *sqlstore.SqlStore, logger *zap.Logger, processor *events.Processor) *Config {
	return &Config{
		db:        db,
		sqlStore:  sqlStore,
		logger:    logger,
		processor: processor,
	}
}

func (c *Config) Deposit(ctx context.Context, virtualaccountid string, userID string) (updated bool, err error) {
	if ctx == nil {
		ctx = context.Background()
	}
	// using the list payment endpoint.
	url := fmt.Sprintf("%s/%s?%s=%s", api, "payments", "virtualNubanId", virtualaccountid)

	if virtualaccountid == "" {
		c.logger.Error("Deposit failed: missing account number")
		return false, ErrEmptyVirtualNuban
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		c.logger.Error("Deposit failed: unable to create new request", zap.Error(err))
		return false, ErrNewRequestFailed
	}
	req.Header.Add("accept", "application/json")
	req.Header.Add("x-anchor-key", apikey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		c.logger.Error("Deposit failed: API connection error", zap.Error(err))
		return false, ErrAPIConnectionFailed
	}
	defer resp.Body.Close()

	apiResponse := paymentResponse{}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.logger.Error("Deposit failed: unable to read response body", zap.Error(err))
		return false, JSONError(err)
	}
	c.logger.Debug("API Response Body", zap.String("body", string(body)))

	if err := json.Unmarshal(body, &apiResponse); err != nil {
		c.logger.Error("Deposit failed: unable to unmarshal JSON response", zap.Error(err))
		return false, JSONError(err)
	}
	c.logger.Debug("Parsed API Response", zap.Any("response", apiResponse))

	updatedAny := false

	paymentData := apiResponse.Data
	for _, data := range paymentData {
		orderID, err := randomgen.GenerateOrderID()
		if err != nil {
			c.logger.Error("Deposit failed: unable to generate order ID", zap.Error(err))
			return updatedAny, err
		}

		transctionID := randomgen.GenerateTransactionID("dep")
		attributes := data.Attributes
		virtualNuban := data.Relationships.VirtualNuban.Data.ID
		deposit := depositID{
			VirtualNuban: virtualNuban,
			ID:           data.ID,
		}

		if err := c.db.SaveDepositID(ctx, deposit); err != nil {
			if err == mongo.ErrDepositIDExist {
				c.logger.Info("Deposit ID already exists, skipping", zap.String("depositID", data.ID))
				continue
			}
			c.logger.Error("Deposit failed: unable to save deposit ID", zap.Error(err))
			return updatedAny, DBConnectionError(err)
		}

		bal, err := c.db.GetBalance(ctx, userID)
		if err != nil {
			c.logger.Error("Deposit failed: unable to fetch balance", zap.Error(err))
			return updatedAny, DBConnectionError(err)
		}
		c.logger.Debug("Fetched Balance", zap.Any("balance", bal))

		breakdown, err := c.CalculateDepositBreakdown(ctx, data.Attributes.Amount)
		if err != nil {
			c.logger.Error("Deposit failed: unable to calculate deposit breakdown", zap.Error(err))
			return updatedAny, err
		}

		newBalance, depositAmount := balance.NewBalanceDeposit(bal, breakdown.NetAmountCredited)
		parsedBalance, _ := primitive.ParseDecimal128(newBalance.String())
		c.logger.Debug("New Balance Calculated", zap.String("newBalance", newBalance.String()))

		userBalance := models.Balance{
			VirtualNuban: virtualNuban,
			Balance:      parsedBalance,
			UserID:       userID,
			UpdatedAt:    time.Now().UTC(),
		}
		if err := c.db.SaveBalance(ctx, userID, userBalance); err != nil {
			c.logger.Error("Deposit failed: unable to save user balance", zap.Error(err))
			return updatedAny, DBConnectionError(err)
		}

		createdAt, err := time.Parse("2006-01-02T15:04:05", data.Attributes.CreatedAt)
		if err != nil {
			c.logger.Error("Deposit failed: unable to parse createdAt", zap.Error(err))
			return updatedAny, err
		}

		result := models.DepositResponse{
			UserID:                 userID,
			Status:                 "success",
			Amount:                 depositAmount.StringFixed(2),
			WalletType:             "Nigerian NGN Wallet",
			Bank_Name:              attributes.CounterParty.Bank.Name,
			Account_Name:           attributes.CounterParty.AccountName,
			Account_No:             attributes.CounterParty.AccountNumber,
			TransactionProduct:     "Virtual Account",
			TransactionDescription: "NGN Wallet Top Up",
			Message:                data.Attributes.Narration,
			Order_ID:               orderID,
			Transaction_ID:         transctionID,
			Reference:              data.Attributes.PaymentReference,
			CreatedAt:              createdAt,
			APICharge:              breakdown.APICharge.StringFixed(2),
			ServiceCharge:          breakdown.ServiceCharge.StringFixed(2),
			GrossAmount:            breakdown.GrossAmount.StringFixed(2),
			ServiceChargeCap:       breakdown.ServiceChargeCap.StringFixed(2),
			ServiceChargeApplied:   breakdown.ServiceChargeApplied.StringFixed(2),
			NetAmountCredited:      breakdown.NetAmountCredited.StringFixed(2),
			ChargeWasCapped:        breakdown.ChargeWasCapped,
		}

		if err := c.saveTransaction(ctx, result); err != nil {
			c.logger.Error("Deposit failed: unable to save transaction", zap.Error(err))
			return updatedAny, DBConnectionError(err)
		}
		c.logger.Info("Deposit transaction saved successfully", zap.Any("transaction", result))

		updatedAny = true

		amount := ""
		if breakdown.GrossAmount.LessThan(decimal.NewFromInt(1)) {
			continue
		} else {
			amount = breakdown.NetAmountCredited.StringFixed(2)

			// Emit wallet.funded event
			if c.processor != nil {
				go func(uID string, amt string, txID string) {
					baseCtx := ctx
					if baseCtx == nil {
						baseCtx = context.Background()
					}
					baseCtx = context.WithoutCancel(baseCtx)

					taskCtx, cancel := context.WithTimeout(baseCtx, 10*time.Second)
					defer cancel()
					ev := &events.Event{
						Type:      "wallet.funded",
						UserID:    uID,
						Amount:    amt,
						TxID:      txID,
						TS:        time.Now().UTC(),
						Published: false,
					}
					if err := c.processor.ProcessEvent(taskCtx, ev); err != nil {
						c.logger.Error("failed to process wallet.funded event", zap.Error(err), zap.String("user_id", uID))
					}

				}(userID, amount, transctionID)
			}
		}
	}

	return updatedAny, nil
}

// write to save transaction to the database
func (c *Config) saveTransaction(ctx context.Context, detail models.DepositResponse) error {
	if ctx == nil {
		ctx = context.Background()
	}
	err := c.db.SaveDeposit(ctx, detail)
	if err != nil {
		return err
	}
	return nil
}
