package txsFee

import (
	"bytes"
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/ElrondNetwork/elrond-go-core/core"
	"github.com/ElrondNetwork/elrond-go-core/data/block"
	"github.com/ElrondNetwork/elrond-go-core/data/smartContractResult"
	"github.com/ElrondNetwork/elrond-go/integrationTests/vm"
	"github.com/ElrondNetwork/elrond-go/integrationTests/vm/txsFee/utils"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/sharding"
	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
	"github.com/stretchr/testify/require"
)

func TestESDTTransferShouldWork(t *testing.T) {
	testContext, err := vm.CreatePreparedTxProcessorWithVMs(vm.ArgEnableEpoch{})
	require.Nil(t, err)
	defer testContext.Close()

	sndAddr := []byte("12345678901234567890123456789012")
	rcvAddr := []byte("12345678901234567890123456789022")

	egldBalance := big.NewInt(100000000)
	esdtBalance := big.NewInt(100000000)
	token := []byte("miiutoken")
	utils.CreateAccountWithESDTBalance(t, testContext.Accounts, sndAddr, egldBalance, token, 0, esdtBalance)

	gasPrice := uint64(10)
	gasLimit := uint64(40)
	tx := utils.CreateESDTTransferTx(0, sndAddr, rcvAddr, token, big.NewInt(100), gasPrice, gasLimit)
	retCode, err := testContext.TxProcessor.ProcessTransaction(tx)
	require.Equal(t, vmcommon.Ok, retCode)
	require.Nil(t, err)
	require.Nil(t, testContext.GetLatestError())

	_, err = testContext.Accounts.Commit()
	require.Nil(t, err)

	expectedBalanceSnd := big.NewInt(99999900)
	utils.CheckESDTBalance(t, testContext, sndAddr, token, expectedBalanceSnd)

	expectedReceiverBalance := big.NewInt(100)
	utils.CheckESDTBalance(t, testContext, rcvAddr, token, expectedReceiverBalance)

	expectedEGLDBalance := big.NewInt(99999640)
	utils.TestAccount(t, testContext.Accounts, sndAddr, 1, expectedEGLDBalance)

	// check accumulated fees
	accumulatedFees := testContext.TxFeeHandler.GetAccumulatedFees()
	require.Equal(t, big.NewInt(360), accumulatedFees)

	intermediateTxs := testContext.GetIntermediateTransactions(t)
	testIndexer := vm.CreateTestIndexer(t, testContext.ShardCoordinator, testContext.EconomicsData)
	testIndexer.SaveTransaction(tx, block.TxBlock, intermediateTxs)

	indexerTx := testIndexer.GetIndexerPreparedTransaction(t)
	require.Equal(t, uint64(36), indexerTx.GasUsed)
	require.Equal(t, "360", indexerTx.Fee)
}

func TestESDTTransferShouldWorkToMuchGasShouldConsumeAllGas(t *testing.T) {
	testContext, err := vm.CreatePreparedTxProcessorWithVMs(vm.ArgEnableEpoch{})
	require.Nil(t, err)
	defer testContext.Close()

	sndAddr := []byte("12345678901234567890123456789012")
	rcvAddr := []byte("12345678901234567890123456789022")

	egldBalance := big.NewInt(100000000)
	esdtBalance := big.NewInt(100000000)
	token := []byte("miiutoken")
	utils.CreateAccountWithESDTBalance(t, testContext.Accounts, sndAddr, egldBalance, token, 0, esdtBalance)

	gasPrice := uint64(10)
	gasLimit := uint64(1000)
	tx := utils.CreateESDTTransferTx(0, sndAddr, rcvAddr, token, big.NewInt(100), gasPrice, gasLimit)
	retCode, err := testContext.TxProcessor.ProcessTransaction(tx)
	require.Equal(t, vmcommon.Ok, retCode)
	require.Nil(t, err)
	require.Nil(t, testContext.GetLatestError())

	_, err = testContext.Accounts.Commit()
	require.Nil(t, err)

	expectedBalanceSnd := big.NewInt(99999900)
	utils.CheckESDTBalance(t, testContext, sndAddr, token, expectedBalanceSnd)

	expectedReceiverBalance := big.NewInt(100)
	utils.CheckESDTBalance(t, testContext, rcvAddr, token, expectedReceiverBalance)

	expectedEGLDBalance := big.NewInt(99990000)
	utils.TestAccount(t, testContext.Accounts, sndAddr, 1, expectedEGLDBalance)

	// check accumulated fees
	accumulatedFees := testContext.TxFeeHandler.GetAccumulatedFees()
	require.Equal(t, big.NewInt(10000), accumulatedFees)

	intermediateTxs := testContext.GetIntermediateTransactions(t)
	testIndexer := vm.CreateTestIndexer(t, testContext.ShardCoordinator, testContext.EconomicsData)
	testIndexer.SaveTransaction(tx, block.TxBlock, intermediateTxs)

	indexerTx := testIndexer.GetIndexerPreparedTransaction(t)
	require.Equal(t, tx.GasLimit, indexerTx.GasUsed)
	require.Equal(t, "10000", indexerTx.Fee)
}

func TestESDTTransferInvalidESDTValueShouldConsumeGas(t *testing.T) {
	testContext, err := vm.CreatePreparedTxProcessorWithVMs(vm.ArgEnableEpoch{})
	require.Nil(t, err)
	defer testContext.Close()

	sndAddr := []byte("12345678901234567890123456789012")
	rcvAddr := []byte("12345678901234567890123456789022")

	egldBalance := big.NewInt(100000000)
	esdtBalance := big.NewInt(100000000)
	token := []byte("miiutoken")
	utils.CreateAccountWithESDTBalance(t, testContext.Accounts, sndAddr, egldBalance, token, 0, esdtBalance)

	gasPrice := uint64(10)
	gasLimit := uint64(1000)
	tx := utils.CreateESDTTransferTx(0, sndAddr, rcvAddr, token, big.NewInt(100000000+1), gasPrice, gasLimit)

	retCode, err := testContext.TxProcessor.ProcessTransaction(tx)
	require.Equal(t, vmcommon.UserError, retCode)
	require.Equal(t, process.ErrFailedTransaction, err)

	_, err = testContext.Accounts.Commit()
	require.Nil(t, err)

	utils.CheckESDTBalance(t, testContext, sndAddr, token, egldBalance)

	expectedReceiverBalance := big.NewInt(0)
	utils.CheckESDTBalance(t, testContext, rcvAddr, token, expectedReceiverBalance)

	expectedEGLDBalance := big.NewInt(99990000)
	utils.TestAccount(t, testContext.Accounts, sndAddr, 1, expectedEGLDBalance)

	// check accumulated fees
	accumulatedFees := testContext.TxFeeHandler.GetAccumulatedFees()
	require.Equal(t, big.NewInt(10000), accumulatedFees)

	intermediateTxs := testContext.GetIntermediateTransactions(t)
	testIndexer := vm.CreateTestIndexer(t, testContext.ShardCoordinator, testContext.EconomicsData)
	testIndexer.SaveTransaction(tx, block.TxBlock, intermediateTxs)

	indexerTx := testIndexer.GetIndexerPreparedTransaction(t)
	require.Equal(t, tx.GasLimit, indexerTx.GasUsed)
	require.Equal(t, "10000", indexerTx.Fee)
}

func TestESDTTransferCallBackOnErrorShouldNotGenerateSCRsFurther(t *testing.T) {
	shardC, _ := sharding.NewMultiShardCoordinator(2, 0)
	testContext, err := vm.CreatePreparedTxProcessorWithVMsWithShardCoordinator(vm.ArgEnableEpoch{}, shardC)
	require.Nil(t, err)
	defer testContext.Close()

	sndAddr := bytes.Repeat([]byte{0}, 32)
	sndAddr[12] = 1
	sndAddr[31] = 1
	rcvAddr := bytes.Repeat([]byte{0}, 32)
	rcvAddr[12] = 1

	egldBalance := big.NewInt(100000000)
	esdtBalance := big.NewInt(100000000)
	token := []byte("miiutoken")
	utils.CreateAccountWithESDTBalance(t, testContext.Accounts, sndAddr, egldBalance, token, 0, esdtBalance)

	hexEncodedToken := hex.EncodeToString(token)
	esdtValueEncoded := hex.EncodeToString(big.NewInt(100).Bytes())
	txDataField := bytes.Join([][]byte{[]byte(core.BuiltInFunctionESDTTransfer), []byte(hexEncodedToken), []byte(esdtValueEncoded), []byte(hex.EncodeToString([]byte("something")))}, []byte("@"))
	scr := &smartContractResult.SmartContractResult{
		Nonce:         0,
		Value:         big.NewInt(0),
		RcvAddr:       rcvAddr,
		SndAddr:       sndAddr,
		Data:          txDataField,
		CallType:      0,
		ReturnMessage: []byte("something"),
	}
	retCode, err := testContext.ScProcessor.ProcessSmartContractResult(scr)
	require.Equal(t, vmcommon.Ok, retCode)
	require.Nil(t, err)
	require.Nil(t, testContext.GetLatestError())

	_, err = testContext.Accounts.Commit()
	require.Nil(t, err)

	expectedReceiverBalance := big.NewInt(100)
	utils.CheckESDTBalance(t, testContext, rcvAddr, token, expectedReceiverBalance)

	// check accumulated fees
	accumulatedFees := testContext.TxFeeHandler.GetAccumulatedFees()
	require.Equal(t, big.NewInt(0), accumulatedFees)

	intermediateTxs := testContext.GetIntermediateTransactions(t)
	require.Equal(t, 0, len(intermediateTxs))
}
