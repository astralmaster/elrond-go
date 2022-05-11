package factory

import (
	"github.com/ElrondNetwork/elrond-go-core/core"
	"github.com/astralmaster/elrond-go/node/external"
	"github.com/astralmaster/elrond-go/node/trieIterators"
	"github.com/astralmaster/elrond-go/node/trieIterators/disabled"
)

// CreateDirectStakedListHandler will create a new instance of DirectStakedListHandler
func CreateDirectStakedListHandler(args trieIterators.ArgTrieIteratorProcessor) (external.DirectStakedListHandler, error) {
	//TODO add unit tests
	if args.ShardID != core.MetachainShardId {
		return disabled.NewDisabledDirectStakedListProcessor(), nil
	}

	return trieIterators.NewDirectStakedListProcessor(args)
}
