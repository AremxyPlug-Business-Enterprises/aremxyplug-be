package randomgen

import (
	"crypto/rand"
	"fmt"
	"math"
	"math/big"
	random "math/rand"
	"os"
	"strconv"
	"strings"
	"time"
)

var (
	charset = os.Getenv("CHARSET")
	numbset = os.Getenv("NUMBSET")
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

func GenerateTransactionID(product string) string {

	prod_type := strings.ToUpper(product)
	seedRand := random.New(random.NewSource(time.Now().UnixNano()))

	b := make([]byte, 5)
	for i := range b {
		b[i] = charset[seedRand.Intn(len(charset))]
	}

	randgen := string(b)

	id := fmt.Sprintf("%s-%s-%s", "AP", prod_type, randgen)

	return id
}

// generateOrderID generates a unique OrderID
func GenerateOrderID() (int, error) {

	numbset, valid := os.LookupEnv("NUMBSET")
	if !valid {
		fmt.Printf("%s", "Could not find NUMBSET environment variable")
		numbset = "1234567890"
	}

	seedRand := random.New(random.NewSource(int64(time.Now().UnixNano())))

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

func GenerateRequestID() string {

	charset, valid := os.LookupEnv("CHARSET")
	if !valid {
		fmt.Printf("%s", "Could not find NUMBSET environment variable")
		charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	}

	random.New(random.NewSource(int64(time.Now().UnixNano())))

	randomChars := make([]byte, 3)
	for i := range randomChars {
		randomChars[i] = charset[random.Intn(len(charset))]
	}

	lagos, _ := time.LoadLocation("Africa/Lagos")

	current_time := time.Now().In(lagos).Format("200601021504")

	return current_time + string(randomChars)
}
