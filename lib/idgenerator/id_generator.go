package idgenerator

import (
	"math/rand"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
)

//go:generate mockgen -source=id_generator.go -destination=../../mocks/id_generator_mock.go -package=mocks
//const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

type IdGenerator interface {
	Generate() string
	GenerateV4UUID() string
}

type ulIdGenerator struct {
	entropy *ulid.MonotonicEntropy
}

func New() IdGenerator {
	entropy := ulid.Monotonic(rand.New(rand.NewSource(time.Now().UnixNano())), 0)
	return &ulIdGenerator{entropy: entropy}
}

func (generator *ulIdGenerator) Generate() string {
	return strings.ToLower(ulid.MustNew(ulid.Timestamp(time.Now()), generator.entropy).String())
}

func (generator *ulIdGenerator) GenerateV4UUID() string {
	return uuid.NewString()
}
