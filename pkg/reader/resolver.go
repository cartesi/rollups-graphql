package reader

import (
	"github.com/cartesi/rollups-graphql/pkg/convenience/services"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

type Resolver struct {
	convenienceService *services.ConvenienceService
	adapter            Adapter
}
