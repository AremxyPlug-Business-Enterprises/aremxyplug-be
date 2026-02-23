package services

import (
	"github.com/aremxyplug-be/lib/services/edu"
	"github.com/aremxyplug-be/lib/services/electric"
	telecom "github.com/aremxyplug-be/lib/services/telcom"
	"github.com/aremxyplug-be/lib/services/tvsub"
)

type ProductServiceImpl struct {
	tvsub.TVSubService
	telecom.TelecomProducts
	edu.Edurecords
	electric.ElectricProducts
}

type ProductService interface {
	tvsub.TVSubService
	telecom.TelecomProducts
	edu.Edurecords
	electric.ElectricProducts
}
