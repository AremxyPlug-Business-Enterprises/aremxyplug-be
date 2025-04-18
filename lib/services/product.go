package services

import (
	telecom "github.com/aremxyplug-be/lib/services/telcom"
	"github.com/aremxyplug-be/lib/services/tvsub"
)

type ProductServiceImpl struct {
	tvsub.TVSubService
	telecom.TelecomProducts
}

type ProductService interface {
	tvsub.TVSubService
	telecom.TelecomProducts
}
