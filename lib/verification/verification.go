package verification

import (
	"context"

	"github.com/aremxyplug-be/db/models"
	"github.com/aremxyplug-be/lib/verification/bvn"
	"github.com/aremxyplug-be/lib/verification/nin"
)

type VerificationClient interface {
	VerifyBVN(ctx context.Context, bvn string, phone string, user models.User) (*bvn.BVNVerificationResult, error)
	VerifyNIN(ctx context.Context, nin string, user models.User) (*nin.NINVerificationResult, error)
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

func (v *verificationClient) VerifyBVN(ctx context.Context, bvn string, phone string, user models.User) (*bvn.BVNVerificationResult, error) {
	return v.bvnConfig.VerifyBVN(ctx, bvn, phone, user)
}

func (v *verificationClient) VerifyNIN(ctx context.Context, nin string, user models.User) (*nin.NINVerificationResult, error) {
	return v.ninConfig.VerifyNIN(ctx, nin, user)
}
