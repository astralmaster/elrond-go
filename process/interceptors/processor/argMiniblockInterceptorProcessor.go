package processor

import (
	"github.com/ElrondNetwork/elrond-go-core/hashing"
	"github.com/ElrondNetwork/elrond-go-core/marshal"
	"github.com/astralmaster/elrond-go/process"
	"github.com/astralmaster/elrond-go/sharding"
	"github.com/astralmaster/elrond-go/storage"
)

// ArgMiniblockInterceptorProcessor is the argument for the interceptor processor used for miniblocks
type ArgMiniblockInterceptorProcessor struct {
	MiniblockCache   storage.Cacher
	Marshalizer      marshal.Marshalizer
	Hasher           hashing.Hasher
	ShardCoordinator sharding.Coordinator
	WhiteListHandler process.WhiteListHandler
}
