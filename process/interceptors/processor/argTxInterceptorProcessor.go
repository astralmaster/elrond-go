package processor

import (
	"github.com/astralmaster/elrond-go/dataRetriever"
	"github.com/astralmaster/elrond-go/process"
)

// ArgTxInterceptorProcessor is the argument for the interceptor processor used for transactions
// (balance txs, smart contract results, reward and so on)
type ArgTxInterceptorProcessor struct {
	ShardedDataCache dataRetriever.ShardedDataCacherNotifier
	TxValidator      process.TxValidator
}
