package txpool

import (
	"github.com/astralmaster/elrond-go/storage"
	"github.com/astralmaster/elrond-go/storage/txcache"
)

type txCache interface {
	storage.Cacher

	AddTx(tx *txcache.WrappedTransaction) (ok bool, added bool)
	GetByTxHash(txHash []byte) (*txcache.WrappedTransaction, bool)
	RemoveTxByHash(txHash []byte) bool
	ImmunizeTxsAgainstEviction(keys [][]byte)
	ForEachTransaction(function txcache.ForEachTransaction)
	NumBytes() int
	Diagnose(deep bool)
}
