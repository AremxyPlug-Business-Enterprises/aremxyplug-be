package deposit

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"
)

type ChargeBreakdown struct {
	APICharge            decimal.Decimal
	ServiceCharge        decimal.Decimal
	GrossAmount          decimal.Decimal
	ServiceChargeCap     decimal.Decimal
	ServiceChargeApplied decimal.Decimal
	NetAmountCredited    decimal.Decimal
	ChargeWasCapped      bool
}

func (c *Config) CalculateDepositBreakdown(ctx context.Context, amountMinor float64) (ChargeBreakdown, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if c == nil || c.sqlStore == nil {
		return ChargeBreakdown{}, fmt.Errorf("deposit configuration is incomplete")
	}

	charges, err := c.sqlStore.GetWalletCharges(ctx)
	if err != nil {
		return ChargeBreakdown{}, err
	}
	if charges.APICharge == nil || charges.ServiceCharge == nil || charges.ServiceChargeCap == nil {
		return ChargeBreakdown{}, fmt.Errorf("wallet charges configuration is incomplete")
	}

	grossAmount := decimal.NewFromFloat(amountMinor).Div(decimal.NewFromInt(100)).Round(2)
	serviceChargeRate := decimal.NewFromFloat(*charges.ServiceCharge)
	serviceChargeCap := decimal.NewFromFloat(*charges.ServiceChargeCap).Round(2)
	serviceChargeApplied := grossAmount.Mul(serviceChargeRate).Div(decimal.NewFromInt(100)).Round(2)
	chargeWasCapped := serviceChargeApplied.GreaterThan(serviceChargeCap)
	if chargeWasCapped {
		serviceChargeApplied = serviceChargeCap
	}
	if serviceChargeApplied.GreaterThan(grossAmount) {
		serviceChargeApplied = grossAmount
	}

	netAmountCredited := grossAmount.Sub(serviceChargeApplied)

	return ChargeBreakdown{
		APICharge:            decimal.NewFromFloat(*charges.APICharge).Round(2),
		ServiceCharge:        decimal.NewFromFloat(*charges.ServiceCharge).Round(2),
		GrossAmount:          grossAmount,
		ServiceChargeCap:     serviceChargeCap,
		ServiceChargeApplied: serviceChargeApplied,
		NetAmountCredited:    netAmountCredited,
		ChargeWasCapped:      chargeWasCapped,
	}, nil
}
