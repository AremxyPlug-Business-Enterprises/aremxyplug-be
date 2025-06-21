package services

import (
	"github.com/aremxyplug-be/lib/services/edu"
	telecom "github.com/aremxyplug-be/lib/services/telcom"
	"github.com/aremxyplug-be/lib/services/tvsub"
)

type ProductServiceImpl struct {
	tvsub.TVSubService
	telecom.TelecomProducts
	edu.Edurecords
}

type ProductService interface {
	tvsub.TVSubService
	telecom.TelecomProducts
	edu.Edurecords
}
