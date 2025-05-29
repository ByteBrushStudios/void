package state

import (
	_ "embed"
	"void/types"

	"github.com/infinitybotlist/eureka/snippets"
	"go.uber.org/zap"
)

var (
	Registry types.ServiceRegistry
	Logger   *zap.SugaredLogger
)

// Init initializes the state
func Init() {
	Logger = snippets.CreateZap()
}
