package balance

import "errors"

func isEnough(balance, payment_value float64) bool {
	return payment_value <= balance
}

func NewBalanceDeposit(balance, deposit float64) (newBalance float64) {
	return balance + deposit
}

func NewBalancePayment(balance, payment float64) (newBalance float64) {
	return balance - payment
}

// should be called before the actual handler for the payment.
func CanPay(balance, amount float64) (bool, error) {

	if !isEnough(balance, amount) {
		return false, errors.New("insufficient balance to carry out the transaction")
	}

	return true, nil
}
