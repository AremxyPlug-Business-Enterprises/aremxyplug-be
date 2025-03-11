package balance

import (
	"errors"

	"github.com/shopspring/decimal"
)

func isEnough(balance, paymentValue decimal.Decimal) bool {
	return paymentValue.LessThanOrEqual(balance)
}

func NewBalanceDeposit(bal float64, deposit float64) (newBalance, depositAmount decimal.Decimal) {
	// Calculate 1% of the deposit amount

	fullDepositAmount := deposit / 100
	depositAmount = decimal.NewFromFloat(fullDepositAmount)

	deduction := decimal.NewFromFloat((0.01 * fullDepositAmount))

	// Subtract 1% of the deposit from the deposit amount
	depositAfterDeduction := depositAmount.Sub(deduction)

	// Add the adjusted deposit amount to the balance
	balance := decimal.NewFromFloat(bal)
	newBalance = balance.Add(depositAfterDeduction)

	return newBalance, depositAfterDeduction
}

func NewBalanceTransfer(balance, transferAmmount decimal.Decimal) (newBalance decimal.Decimal) {

	deduction := 50.00

	return balance.Sub(transferAmmount).Sub(decimal.NewFromFloat(deduction))
}

func NewBalancePayment(balance, payment decimal.Decimal) (newBalance decimal.Decimal) {
	return balance.Sub(payment)
}

// should be called before the actual handler for the payment.

func CanPay(balance, amount decimal.Decimal) (bool, error) {
	if !isEnough(balance, amount) {
		return false, errors.New("insufficient balance to carry out the transaction")
	}

	return true, nil
}

func CanTransfer(balance, amountToTransfer decimal.Decimal) (bool, error) {
	totalAmountCharged := amountToTransfer.Add(decimal.NewFromFloat(50.00))

	if balance.LessThan(totalAmountCharged) {
		return false, errors.New("insufficient balance to carry out the transfer")
	}

	return true, nil
}
