package postprocess

import (
	"sync"

	"github.com/ElrondNetwork/elrond-go-core/core"
	"github.com/ElrondNetwork/elrond-go-core/core/check"
	"github.com/ElrondNetwork/elrond-go-core/data"
	"github.com/ElrondNetwork/elrond-go-core/data/block"
	"github.com/ElrondNetwork/elrond-go-core/hashing"
	"github.com/ElrondNetwork/elrond-go-core/marshal"
	"github.com/ElrondNetwork/elrond-go-logger"
	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/sharding"
)

var _ process.DataMarshalizer = (*basePostProcessor)(nil)

var log = logger.GetOrCreate("process/block/postprocess")

type basePostProcessor struct {
	hasher           hashing.Hasher
	marshalizer      marshal.Marshalizer
	store            dataRetriever.StorageService
	shardCoordinator sharding.Coordinator
	storageType      dataRetriever.UnitType

	mutInterResultsForBlock sync.RWMutex
	interResultsForBlock    map[string]*process.TxInfo
	mapProcessedResult      map[string]struct{}
	intraShardMiniBlock     *block.MiniBlock
	economicsFee            process.FeeHandler
	postProcessorTxsHandler process.PostProcessorTxsHandler
}

// SaveCurrentIntermediateTxToStorage saves all current intermediate results to the provided storage unit
func (bpp *basePostProcessor) SaveCurrentIntermediateTxToStorage() error {
	bpp.mutInterResultsForBlock.RLock()
	defer bpp.mutInterResultsForBlock.RUnlock()

	for _, txInfoValue := range bpp.interResultsForBlock {
		if check.IfNil(txInfoValue.Tx) {
			return process.ErrMissingTransaction
		}

		buff, err := bpp.marshalizer.Marshal(txInfoValue.Tx)
		if err != nil {
			return err
		}

		errNotCritical := bpp.store.Put(bpp.storageType, bpp.hasher.Compute(string(buff)), buff)
		if errNotCritical != nil {
			log.Debug("SaveCurrentIntermediateTxToStorage put", "type", bpp.storageType, "error", errNotCritical.Error())
		}
	}

	return nil
}

// CreateBlockStarted cleans the local cache map for processed/created intermediate transactions at this round
func (bpp *basePostProcessor) CreateBlockStarted() {
	bpp.mutInterResultsForBlock.Lock()
	bpp.interResultsForBlock = make(map[string]*process.TxInfo)
	bpp.intraShardMiniBlock = nil
	bpp.mapProcessedResult = make(map[string]struct{})
	bpp.mutInterResultsForBlock.Unlock()
}

// CreateMarshalizedData creates the marshalized data for broadcasting purposes
func (bpp *basePostProcessor) CreateMarshalizedData(txHashes [][]byte) ([][]byte, error) {
	bpp.mutInterResultsForBlock.RLock()
	defer bpp.mutInterResultsForBlock.RUnlock()

	mrsTxs := make([][]byte, 0, len(txHashes))
	for _, txHash := range txHashes {
		txInfoObject := bpp.interResultsForBlock[string(txHash)]
		if txInfoObject == nil || check.IfNil(txInfoObject.Tx) {
			log.Warn("basePostProcessor.CreateMarshalizedData: tx not found", "hash", txHash)
			continue
		}

		txMrs, err := bpp.marshalizer.Marshal(txInfoObject.Tx)
		if err != nil {
			return nil, process.ErrMarshalWithoutSuccess
		}
		mrsTxs = append(mrsTxs, txMrs)
	}

	return mrsTxs, nil
}

// GetAllCurrentFinishedTxs returns the cached finalized transactions for current round
func (bpp *basePostProcessor) GetAllCurrentFinishedTxs() map[string]data.TransactionHandler {
	bpp.mutInterResultsForBlock.RLock()

	scrPool := make(map[string]data.TransactionHandler)
	for txHash, txInfoInterResult := range bpp.interResultsForBlock {
		scrPool[txHash] = txInfoInterResult.Tx
	}
	bpp.mutInterResultsForBlock.RUnlock()

	return scrPool
}

func (bpp *basePostProcessor) verifyMiniBlock(createMBs map[uint32][]*block.MiniBlock, mb *block.MiniBlock) error {
	createdScrMbs, ok := createMBs[mb.ReceiverShardID]
	if !ok {
		log.Debug("missing mini block", "type", mb.Type, "sender", mb.SenderShardID, "receiver", mb.ReceiverShardID, "numTxs", len(mb.TxHashes))
		return process.ErrNilMiniBlocks
	}

	mapCreatedHashes := make(map[string]struct{})
	for _, createdScrMb := range createdScrMbs {
		createdHash, err := core.CalculateHash(bpp.marshalizer, bpp.hasher, createdScrMb)
		if err != nil {
			return err
		}

		mapCreatedHashes[string(createdHash)] = struct{}{}
	}

	receivedHash, err := core.CalculateHash(bpp.marshalizer, bpp.hasher, mb)
	if err != nil {
		return err
	}

	_, existsHash := mapCreatedHashes[string(receivedHash)]
	if !existsHash {
		log.Debug("received mini block", "type", mb.Type, "sender", mb.SenderShardID, "receiver", mb.ReceiverShardID, "numTxs", len(mb.TxHashes))
		for _, createdScrMb := range createdScrMbs {
			log.Debug("created mini block", "type", createdScrMb.Type, "sender", createdScrMb.SenderShardID, "receiver", createdScrMb.ReceiverShardID, "numTxs", len(createdScrMb.TxHashes))
		}
		return process.ErrMiniBlockHashMismatch
	}

	return nil
}

// GetCreatedInShardMiniBlock returns a clone of the intra shard mini block
func (bpp *basePostProcessor) GetCreatedInShardMiniBlock() *block.MiniBlock {
	bpp.mutInterResultsForBlock.RLock()
	defer bpp.mutInterResultsForBlock.RUnlock()

	if bpp.intraShardMiniBlock == nil {
		return nil
	}

	return bpp.intraShardMiniBlock.Clone()
}

// RemoveProcessedResults will remove the processed results since the last init
func (bpp *basePostProcessor) RemoveProcessedResults() {
	bpp.mutInterResultsForBlock.Lock()
	defer bpp.mutInterResultsForBlock.Unlock()

	for txHash := range bpp.mapProcessedResult {
		delete(bpp.interResultsForBlock, txHash)
	}
}

// InitProcessedResults will initialize the processed results
func (bpp *basePostProcessor) InitProcessedResults() {
	bpp.mutInterResultsForBlock.Lock()
	defer bpp.mutInterResultsForBlock.Unlock()

	bpp.mapProcessedResult = make(map[string]struct{})
}

func (bpp *basePostProcessor) splitMiniBlocksIfNeeded(miniBlocks []*block.MiniBlock) []*block.MiniBlock {
	splitMiniBlocks := make([]*block.MiniBlock, 0)
	for _, miniBlock := range miniBlocks {
		if miniBlock.ReceiverShardID == bpp.shardCoordinator.SelfId() {
			splitMiniBlocks = append(splitMiniBlocks, miniBlock)
			continue
		}

		splitMiniBlocks = append(splitMiniBlocks, bpp.splitMiniBlockIfNeeded(miniBlock)...)
	}
	return splitMiniBlocks
}

func (bpp *basePostProcessor) splitMiniBlockIfNeeded(miniBlock *block.MiniBlock) []*block.MiniBlock {
	splitMiniBlocks := make([]*block.MiniBlock, 0)
	currentMiniBlock := createEmptyMiniBlock(miniBlock)
	gasLimitInReceiverShard := uint64(0)

	for _, txHash := range miniBlock.TxHashes {
		interResult, ok := bpp.interResultsForBlock[string(txHash)]
		if !ok {
			log.Warn("basePostProcessor.splitMiniBlockIfNeeded: missing tx", "hash", txHash)
			currentMiniBlock.TxHashes = append(currentMiniBlock.TxHashes, txHash)
			continue
		}

		isGasLimitExceeded := gasLimitInReceiverShard+interResult.Tx.GetGasLimit() >
			bpp.economicsFee.MaxGasLimitPerMiniBlockForSafeCrossShard()
		if isGasLimitExceeded {
			log.Debug("basePostProcessor.splitMiniBlockIfNeeded: gas limit exceeded",
				"mb type", currentMiniBlock.Type,
				"sender shard", currentMiniBlock.SenderShardID,
				"receiver shard", currentMiniBlock.ReceiverShardID,
				"initial num txs", len(miniBlock.TxHashes),
				"adjusted num txs", len(currentMiniBlock.TxHashes),
			)

			if len(currentMiniBlock.TxHashes) > 0 {
				splitMiniBlocks = append(splitMiniBlocks, currentMiniBlock)
			}

			currentMiniBlock = createEmptyMiniBlock(miniBlock)
			gasLimitInReceiverShard = 0
		}

		gasLimitInReceiverShard += interResult.Tx.GetGasLimit()
		currentMiniBlock.TxHashes = append(currentMiniBlock.TxHashes, txHash)
	}

	if len(currentMiniBlock.TxHashes) > 0 {
		splitMiniBlocks = append(splitMiniBlocks, currentMiniBlock)
	}

	return splitMiniBlocks
}

func createEmptyMiniBlock(miniBlock *block.MiniBlock) *block.MiniBlock {
	return &block.MiniBlock{
		SenderShardID:   miniBlock.SenderShardID,
		ReceiverShardID: miniBlock.ReceiverShardID,
		Type:            miniBlock.Type,
		Reserved:        miniBlock.Reserved,
		TxHashes:        make([][]byte, 0),
	}
}

func createMiniBlocksMap(scrMbs []*block.MiniBlock) map[uint32][]*block.MiniBlock {
	createdMapMbs := make(map[uint32][]*block.MiniBlock)
	for _, mb := range scrMbs {
		_, ok := createdMapMbs[mb.ReceiverShardID]
		if !ok {
			createdMapMbs[mb.ReceiverShardID] = make([]*block.MiniBlock, 0)
		}

		createdMapMbs[mb.ReceiverShardID] = append(createdMapMbs[mb.ReceiverShardID], mb)
	}

	return createdMapMbs
}

// GetProcessedResults gets all the intermediate transactions generated since the last init
func (bpp *basePostProcessor) GetProcessedResults() map[uint32][]*process.TxInfo {
	bpp.mutInterResultsForBlock.RLock()
	defer bpp.mutInterResultsForBlock.RUnlock()

	mapIntermediateTxs := make(map[uint32][]*process.TxInfo)

	for intermediateTxHash := range bpp.mapProcessedResult {
		txInfo, ok := bpp.interResultsForBlock[intermediateTxHash]
		if !ok {
			log.Error("basePostProcessor.GetProcessedResults",
				"scr hash", intermediateTxHash,
				"error", process.ErrSmartContractResultNotFound)
			continue
		}

		txInfo.TxHash = []byte(intermediateTxHash)

		if _, ok = mapIntermediateTxs[txInfo.ReceiverShardID]; !ok {
			mapIntermediateTxs[txInfo.ReceiverShardID] = make([]*process.TxInfo, 0)
		}

		mapIntermediateTxs[txInfo.ReceiverShardID] = append(mapIntermediateTxs[txInfo.ReceiverShardID], txInfo)
	}

	return mapIntermediateTxs
}

func (bpp *basePostProcessor) filterPostProcessMiniBlocks(miniBlocks block.MiniBlockSlice) block.MiniBlockSlice {
	finalMiniBlocks := make(block.MiniBlockSlice, 0)
	for i := 0; i < len(miniBlocks); i++ {
		txHashes := make([][]byte, 0)
		for _, txHash := range miniBlocks[i].TxHashes {
			if bpp.postProcessorTxsHandler.IsPostProcessorTxAdded(txHash) {
				continue
			}
			txHashes = append(txHashes, txHash)
		}

		//TODO: This should be move on log.Trace
		log.Debug("DEBUGGING: basePostProcessor.filterPostProcessMiniBlocks", "mb type", miniBlocks[i].Type,
			"sender", miniBlocks[i].SenderShardID,
			"receiver", miniBlocks[i].ReceiverShardID,
			"num txs in mb", len(miniBlocks[i].TxHashes),
			"num txs added", len(txHashes))

		if len(txHashes) == 0 {
			continue
		}

		miniBlocks[i].TxHashes = txHashes
		finalMiniBlocks = append(finalMiniBlocks, miniBlocks[i])
	}

	return finalMiniBlocks
}
