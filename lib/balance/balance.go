package balance

import "errors"

func isEnough(balance, payment_value float64) bool {
	return payment_value <= balance
}

func NewBalanceDeposit(balance, deposit float64) (newBalance, depositAmount float64) {
	// Calculate 1% of the deposit amount

	fullDepositAmount := deposit / 100

	deduction := 0.01 * fullDepositAmount

	// Subtract 1% of the deposit from the deposit amount
	depositAfterDeduction := fullDepositAmount - deduction

	// Add the adjusted deposit amount to the balance
	newBalance = balance + depositAfterDeduction

	return newBalance, depositAfterDeduction
}

func NewBalanceTransfer(balance, transferAmmount float64) (newBalance float64) {

	deduction := 50.00

	return balance - transferAmmount - deduction
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

func CanTransfer(balance, amountToTransfer float64) (bool, error) {

	totalAmountCharged := amountToTransfer + 50.00

	if balance < totalAmountCharged {
		return false, errors.New("insufficient balance to carry out the transfer")
	}

	return true, nil

}
