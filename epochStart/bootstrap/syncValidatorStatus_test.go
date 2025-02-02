package bootstrap

import (
	"context"
	"testing"

	"github.com/ElrondNetwork/elrond-go-core/core"
	"github.com/ElrondNetwork/elrond-go-core/core/check"
	"github.com/ElrondNetwork/elrond-go-core/data"
	"github.com/ElrondNetwork/elrond-go-core/data/block"
	"github.com/ElrondNetwork/elrond-go-core/data/endProcess"
	"github.com/ElrondNetwork/elrond-go/epochStart/mock"
	"github.com/ElrondNetwork/elrond-go/sharding"
	"github.com/ElrondNetwork/elrond-go/storage"
	"github.com/ElrondNetwork/elrond-go/testscommon"
	epochStartMocks "github.com/ElrondNetwork/elrond-go/testscommon/bootstrapMocks/epochStart"
	dataRetrieverMock "github.com/ElrondNetwork/elrond-go/testscommon/dataRetriever"
	"github.com/ElrondNetwork/elrond-go/testscommon/hashingMocks"
	"github.com/ElrondNetwork/elrond-go/testscommon/nodeTypeProviderMock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const initRating = uint32(50)

func TestNewSyncValidatorStatus_ShouldWork(t *testing.T) {
	t.Parallel()

	args := getSyncValidatorStatusArgs()
	svs, err := NewSyncValidatorStatus(args)
	require.NoError(t, err)
	require.False(t, check.IfNil(svs))
}

func TestSyncValidatorStatus_NodesConfigFromMetaBlock(t *testing.T) {
	t.Parallel()

	args := getSyncValidatorStatusArgs()
	svs, _ := NewSyncValidatorStatus(args)

	currMb := &block.MetaBlock{
		Nonce: 37,
		Epoch: 0,
		MiniBlockHeaders: []block.MiniBlockHeader{
			{
				Hash:            []byte("mb0-hash"),
				ReceiverShardID: 0,
				SenderShardID:   0,
				Type:            block.TxBlock,
				TxCount:         0,
			},
		},
		EpochStart: block.EpochStart{
			LastFinalizedHeaders: []block.EpochStartShardData{
				{
					ShardID:                 0,
					Epoch:                   0,
					Round:                   0,
					Nonce:                   0,
					HeaderHash:              []byte("hash"),
					RootHash:                []byte("rootHash"),
					FirstPendingMetaBlock:   []byte("hash"),
					LastFinishedMetaBlock:   []byte("hash"),
					PendingMiniBlockHeaders: nil,
				},
			},
		}}
	prevMb := &block.MetaBlock{
		Nonce: 36,
		Epoch: 0,
		MiniBlockHeaders: []block.MiniBlockHeader{
			{
				Hash:            []byte("mb0-hash"),
				ReceiverShardID: 0,
				SenderShardID:   0,
				Type:            block.TxBlock,
				TxCount:         0,
			},
		},
		EpochStart: block.EpochStart{
			LastFinalizedHeaders: []block.EpochStartShardData{
				{
					ShardID:                 0,
					Epoch:                   0,
					Round:                   0,
					Nonce:                   0,
					HeaderHash:              []byte("hash"),
					RootHash:                []byte("rootHash"),
					FirstPendingMetaBlock:   []byte("hash"),
					LastFinishedMetaBlock:   []byte("hash"),
					PendingMiniBlockHeaders: nil,
				},
			},
		},
	}

	registry, _, err := svs.NodesConfigFromMetaBlock(currMb, prevMb)
	require.NoError(t, err)
	require.NotNil(t, registry)
}

func TestSyncValidatorStatus_processValidatorChangesFor(t *testing.T) {
	t.Parallel()

	mbHeaderHash1 := []byte("mb-hash1")
	mbHeaderHash2 := []byte("mb-hash2")

	metaBlock := &block.MetaBlock{
		Nonce: 10,
		Epoch: 1,
		MiniBlockHeaders: []block.MiniBlockHeader{
			{
				Hash: mbHeaderHash1,
				Type: block.TxBlock,
			},
			{
				Hash: mbHeaderHash2,
				Type: block.PeerBlock,
			},
		},
	}

	mb := &block.MiniBlock{
		ReceiverShardID: 1,
		SenderShardID:   0,
	}
	expectedBody := &block.Body{
		MiniBlocks: []*block.MiniBlock{
			mb,
		},
	}

	args := getSyncValidatorStatusArgs()
	svs, _ := NewSyncValidatorStatus(args)

	wasCalled := false
	svs.nodeCoordinator = &mock.NodesCoordinatorStub{
		EpochStartPrepareCalled: func(metaHdr data.HeaderHandler, body data.BodyHandler) {
			wasCalled = true
			assert.Equal(t, metaBlock, metaHdr)
			assert.Equal(t, expectedBody, body)
		},
	}
	svs.miniBlocksSyncer = &epochStartMocks.PendingMiniBlockSyncHandlerStub{
		SyncPendingMiniBlocksCalled: func(miniBlockHeaders []data.MiniBlockHeaderHandler, ctx context.Context) error {
			return nil
		},
		GetMiniBlocksCalled: func() (map[string]*block.MiniBlock, error) {
			return map[string]*block.MiniBlock{
				string(mbHeaderHash2): mb,
			}, nil
		},
	}

	err := svs.processValidatorChangesFor(metaBlock)
	require.NoError(t, err)
	assert.True(t, wasCalled)
}

func TestSyncValidatorStatus_findPeerMiniBlockHeaders(t *testing.T) {
	t.Parallel()

	mbHeader1 := block.MiniBlockHeader{
		Hash: []byte("mb-hash1"),
		Type: block.TxBlock,
	}
	mbHeader2 := block.MiniBlockHeader{
		Hash: []byte("mb-hash2"),
		Type: block.PeerBlock,
	}

	metaBlock := &block.MetaBlock{
		Nonce: 37,
		Epoch: 0,
		MiniBlockHeaders: []block.MiniBlockHeader{
			mbHeader1,
			mbHeader2,
		},
	}

	expectedMbHeaders := []data.MiniBlockHeaderHandler{
		&mbHeader2,
	}

	mbHeaderHandlers := findPeerMiniBlockHeaders(metaBlock)
	require.Equal(t, expectedMbHeaders, mbHeaderHandlers)
}

func TestSyncValidatorStatus_getPeerBlockBodyForMeta(t *testing.T) {
	t.Parallel()

	args := getSyncValidatorStatusArgs()

	mbHeaderHash1 := []byte("mb-hash1")
	mbHeaderHash2 := []byte("mb-hash2")

	metaBlock := &block.MetaBlock{
		Nonce: 37,
		Epoch: 0,
		MiniBlockHeaders: []block.MiniBlockHeader{
			{
				Hash: mbHeaderHash1,
				Type: block.TxBlock,
			},
			{
				Hash: mbHeaderHash2,
				Type: block.PeerBlock,
			},
		},
	}

	svs, _ := NewSyncValidatorStatus(args)
	svs.miniBlocksSyncer = &epochStartMocks.PendingMiniBlockSyncHandlerStub{
		SyncPendingMiniBlocksCalled: func(miniBlockHeaders []data.MiniBlockHeaderHandler, ctx context.Context) error {
			return nil
		},
		GetMiniBlocksCalled: func() (map[string]*block.MiniBlock, error) {
			return map[string]*block.MiniBlock{
				string(mbHeaderHash2): {
					ReceiverShardID: 1,
					SenderShardID:   0,
				},
			}, nil
		},
	}

	expectedBody := &block.Body{
		MiniBlocks: []*block.MiniBlock{
			{
				ReceiverShardID: 1,
				SenderShardID:   0,
			},
		},
	}

	body, err := svs.getPeerBlockBodyForMeta(metaBlock)
	require.NoError(t, err)
	require.Equal(t, expectedBody, body)
}

func getSyncValidatorStatusArgs() ArgsNewSyncValidatorStatus {
	return ArgsNewSyncValidatorStatus{
		DataPool: &dataRetrieverMock.PoolsHolderStub{
			MiniBlocksCalled: func() storage.Cacher {
				return testscommon.NewCacherStub()
			},
		},
		Marshalizer:    &mock.MarshalizerMock{},
		Hasher:         &hashingMocks.HasherMock{},
		RequestHandler: &testscommon.RequestHandlerStub{},
		ChanceComputer: &mock.NodesCoordinatorStub{},
		GenesisNodesConfig: &mock.NodesSetupStub{
			NumberOfShardsCalled: func() uint32 {
				return 1
			},
			InitialNodesInfoForShardCalled: func(shardID uint32) ([]sharding.GenesisNodeInfoHandler, []sharding.GenesisNodeInfoHandler, error) {
				if shardID == core.MetachainShardId {
					return []sharding.GenesisNodeInfoHandler{
							mock.NewNodeInfo([]byte("addr0"), []byte("pubKey0"), core.MetachainShardId, initRating),
							mock.NewNodeInfo([]byte("addr1"), []byte("pubKey1"), core.MetachainShardId, initRating),
						},
						[]sharding.GenesisNodeInfoHandler{&mock.NodeInfoMock{}},
						nil
				}
				return []sharding.GenesisNodeInfoHandler{
						mock.NewNodeInfo([]byte("addr0"), []byte("pubKey0"), 0, initRating),
						mock.NewNodeInfo([]byte("addr1"), []byte("pubKey1"), 0, initRating),
					},
					[]sharding.GenesisNodeInfoHandler{&mock.NodeInfoMock{}},
					nil
			},
			InitialNodesInfoCalled: func() (map[uint32][]sharding.GenesisNodeInfoHandler, map[uint32][]sharding.GenesisNodeInfoHandler) {
				return map[uint32][]sharding.GenesisNodeInfoHandler{
						0: {
							mock.NewNodeInfo([]byte("addr0"), []byte("pubKey0"), 0, initRating),
							mock.NewNodeInfo([]byte("addr1"), []byte("pubKey1"), 0, initRating),
						},
						core.MetachainShardId: {
							mock.NewNodeInfo([]byte("addr0"), []byte("pubKey0"), core.MetachainShardId, initRating),
							mock.NewNodeInfo([]byte("addr1"), []byte("pubKey1"), core.MetachainShardId, initRating),
						},
					}, map[uint32][]sharding.GenesisNodeInfoHandler{0: {
						mock.NewNodeInfo([]byte("addr2"), []byte("pubKey2"), 0, initRating),
						mock.NewNodeInfo([]byte("addr3"), []byte("pubKey3"), 0, initRating),
					}}
			},
			GetShardConsensusGroupSizeCalled: func() uint32 {
				return 2
			},
			GetMetaConsensusGroupSizeCalled: func() uint32 {
				return 2
			},
		},
		NodeShuffler:      &mock.NodeShufflerMock{},
		PubKey:            []byte("public key"),
		ShardIdAsObserver: 0,
		ChanNodeStop:      endProcess.GetDummyEndProcessChannel(),
		NodeTypeProvider:  &nodeTypeProviderMock.NodeTypeProviderStub{},
		IsFullArchive:     false,
	}
}
