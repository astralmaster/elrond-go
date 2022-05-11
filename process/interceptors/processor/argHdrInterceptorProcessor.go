package processor

import (
	"github.com/astralmaster/elrond-go/dataRetriever"
	"github.com/astralmaster/elrond-go/process"
)

// ArgHdrInterceptorProcessor is the argument for the interceptor processor used for headers (shard, meta and so on)
type ArgHdrInterceptorProcessor struct {
	Headers        dataRetriever.HeadersPool
	BlockBlackList process.TimeCacher
}
