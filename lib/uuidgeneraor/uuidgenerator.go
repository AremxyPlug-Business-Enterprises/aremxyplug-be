package uuidgenerator

import "github.com/google/uuid"

//go:generate mockgen -source=uuidgenerator.go -destination=../../mocks/uuidgenerator_mock.go -package=mocks
type UUIDGenerator interface {
	Generate() string
	GenerateFromString(data string) (string, error)
}

// Ensure implements interface
var _ UUIDGenerator = &GoogleUUIDGenerator{}

type GoogleUUIDGenerator struct{}

func (g GoogleUUIDGenerator) Generate() string {
	return uuid.NewString()
}

func (g GoogleUUIDGenerator) GenerateFromString(data string) (string, error) {
	var bytes [16]byte
	// Copy string into bytes array of length 16
	copy(bytes[:], data)
	newUUID, err := uuid.FromBytes(bytes[:])
	if err != nil {
		return "", err
	}

	return newUUID.String(), nil
}

func NewGoogleUUIDGenerator() *GoogleUUIDGenerator {
	return &GoogleUUIDGenerator{}
}
