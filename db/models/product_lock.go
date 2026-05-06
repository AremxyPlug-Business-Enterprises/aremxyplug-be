package models

import "time"

const (
	AllProductsPurchasesEnabled = "ALL_PRODUCTS_PURCHASES_ENABLED"
	AirtimePurchasesEnabled     = "AIRTIME_PURCHASES_ENABLED"
	DataPurchasesEnabled        = "DATA_PURCHASES_ENABLED"
	TVPurchasesEnabled          = "TV_PURCHASES_ENABLED"
	ElectricityPurchasesEnabled = "ELECTRICITY_PURCHASES_ENABLED"
	EduPinsPurchasesEnabled     = "EDU_PINS_PURCHASES_ENABLED"
)

const (
	ProductLockReasonGlobalLock      = "GLOBAL_LOCK"
	ProductLockReasonProductLock     = "PRODUCT_LOCK"
	ProductLockReasonItemUnavailable = "ITEM_UNAVAILABLE"
)

type ProductLock struct {
	Key         string    `json:"key"`
	Enabled     bool      `json:"enabled"`
	Description string    `json:"description"`
	UpdatedAt   time.Time `json:"updated_at"`
}
