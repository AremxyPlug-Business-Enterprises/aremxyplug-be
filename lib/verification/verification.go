package verification

import (
	"github.com/aremxyplug-be/db/models"
	"github.com/aremxyplug-be/lib/verification/bvn"
	"github.com/aremxyplug-be/lib/verification/nin"
)

type VerificationClient interface {
	VerifyBVN(bvn string, user models.User) (*bvn.BVNVerificationResult, error)
	VerifyNIN(nin string, user models.User) (*nin.NINVerificationResult, error)
}

type verificationClient struct {
	bvnConfig *bvn.BvnConfig
	ninConfig *nin.NINConfig
}

func NewVerificationClient(bvnConfig *bvn.BvnConfig, ninConfig *nin.NINConfig) VerificationClient {
	return &verificationClient{
		bvnConfig: bvnConfig,
		ninConfig: ninConfig,
	}
}

func (v *verificationClient) VerifyBVN(bvn string, user models.User) (*bvn.BVNVerificationResult, error) {
	return v.bvnConfig.VerifyBVN(bvn, user)
}

func (v *verificationClient) VerifyNIN(nin string, user models.User) (*nin.NINVerificationResult, error) {
	return v.ninConfig.VerifyNIN(nin, user)
}
