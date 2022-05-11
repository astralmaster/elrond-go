package process

import (
	"math/big"

	"github.com/ElrondNetwork/elrond-go-core/core"
	"github.com/ElrondNetwork/elrond-go-core/data"
	"github.com/ElrondNetwork/elrond-go-core/data/typeConverters"
	"github.com/ElrondNetwork/elrond-go-core/hashing"
	"github.com/ElrondNetwork/elrond-go-core/marshal"
	"github.com/ElrondNetwork/elrond-go-crypto"
	"github.com/astralmaster/elrond-go/common"
	"github.com/astralmaster/elrond-go/config"
	"github.com/astralmaster/elrond-go/dataRetriever"
	"github.com/astralmaster/elrond-go/genesis"
	"github.com/astralmaster/elrond-go/process"
	"github.com/astralmaster/elrond-go/sharding"
	"github.com/astralmaster/elrond-go/state"
	"github.com/astralmaster/elrond-go/update"
)

type coreComponentsHandler interface {
	InternalMarshalizer() marshal.Marshalizer
	TxMarshalizer() marshal.Marshalizer
	Hasher() hashing.Hasher
	AddressPubKeyConverter() core.PubkeyConverter
	Uint64ByteSliceConverter() typeConverters.Uint64ByteSliceConverter
	ChainID() string
	IsInterfaceNil() bool
}

type dataComponentsHandler interface {
	StorageService() dataRetriever.StorageService
	Blockchain() data.ChainHandler
	Datapool() dataRetriever.PoolsHolder
	SetBlockchain(chain data.ChainHandler)
	Clone() interface{}
	IsInterfaceNil() bool
}

// ArgsGenesisBlockCreator holds the arguments which are needed to create a genesis block
type ArgsGenesisBlockCreator struct {
	GenesisTime          uint64
	StartEpochNum        uint32
	Data                 dataComponentsHandler
	Core                 coreComponentsHandler
	Accounts             state.AccountsAdapter
	ValidatorAccounts    state.AccountsAdapter
	InitialNodesSetup    genesis.InitialNodesHandler
	Economics            process.EconomicsDataHandler
	ShardCoordinator     sharding.Coordinator
	AccountsParser       genesis.AccountsParser
	SmartContractParser  genesis.InitialSmartContractParser
	GasSchedule          core.GasScheduleNotifier
	TxLogsProcessor      process.TransactionLogProcessor
	VirtualMachineConfig config.VirtualMachineConfig
	HardForkConfig       config.HardforkConfig
	TrieStorageManagers  map[string]common.StorageManager
	SystemSCConfig       config.SystemSmartContractsConfig
	EpochConfig          *config.EpochConfig
	ImportStartHandler   update.ImportStartHandler
	WorkingDir           string
	BlockSignKeyGen      crypto.KeyGenerator

	GenesisNodePrice *big.Int
	GenesisString    string
	// created components
	importHandler update.ImportHandler
}
