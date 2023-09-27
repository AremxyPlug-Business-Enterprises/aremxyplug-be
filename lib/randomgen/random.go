package randomgen

import (
	"crypto/rand"
	"math"
	"math/big"
	random "math/rand"
	"os"
	"strconv"
	"time"
)

func GenerateRandomNum(numberOfDigits int) (int, error) {
	maxLimit := int64(int(math.Pow10(numberOfDigits)) - 1)
	lowLimit := int(math.Pow10(numberOfDigits - 1))

	randomNumber, err := rand.Int(rand.Reader, big.NewInt(maxLimit))
	if err != nil {
		return 0, err
	}
	randomNumberInt := int(randomNumber.Int64())

	// Handling integers between 0, 10^(n-1) .. for n=4, handling cases between (0, 999)
	if randomNumberInt <= lowLimit {
		randomNumberInt += lowLimit
	}

	// Never likely to occur, kust for safe side.
	if randomNumberInt > int(maxLimit) {
		randomNumberInt = int(maxLimit)
	}
	return randomNumberInt, nil
}

func GenerateTransactionID() string {
	seedRand := random.New(random.NewSource(time.Now().UnixNano()))
	charset := os.Getenv("CHARSET")

	b := make([]byte, 10)
	for i := range b {
		b[i] = charset[seedRand.Intn(len(charset))]
	}

	return string(b)
}

// generateOrderID generates a unique OrderID
func GenerateOrderID() (int, error) {
	seedRand := random.New(random.NewSource(int64(time.Now().UnixNano())))
	numbset := os.Getenv(("NUMBSET"))

	b := make([]byte, 10)
	for i := range b {
		b[i] = numbset[seedRand.Intn(len(numbset))]
	}

	s := string(b)

	Id, err := strconv.Atoi(s)
	if err != nil {
		return 0, err
	}

	return Id, nil
}
