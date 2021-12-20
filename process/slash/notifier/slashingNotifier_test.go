package notifier_test

import (
	"errors"
	"fmt"
	"math/big"
	"testing"

	"github.com/ElrondNetwork/elrond-go-core/core/atomic"
	coreSlash "github.com/ElrondNetwork/elrond-go-core/data/slash"
	"github.com/ElrondNetwork/elrond-go-core/data/transaction"
	crypto "github.com/ElrondNetwork/elrond-go-crypto"
	"github.com/ElrondNetwork/elrond-go/consensus/mock"
	mockGenesis "github.com/ElrondNetwork/elrond-go/genesis/mock"
	mockIntegration "github.com/ElrondNetwork/elrond-go/integrationTests/mock"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/process/slash/notifier"
	"github.com/ElrondNetwork/elrond-go/state"
	"github.com/ElrondNetwork/elrond-go/testscommon"
	"github.com/ElrondNetwork/elrond-go/testscommon/cryptoMocks"
	"github.com/ElrondNetwork/elrond-go/testscommon/hashingMocks"
	"github.com/ElrondNetwork/elrond-go/testscommon/slashMocks"
	stateMock "github.com/ElrondNetwork/elrond-go/testscommon/state"
	"github.com/ElrondNetwork/elrond-go/update"
	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
	"github.com/stretchr/testify/require"
)

func TestNewSlashingNotifier(t *testing.T) {
	tests := []struct {
		args        func() *notifier.SlashingNotifierArgs
		expectedErr error
	}{
		{
			args: func() *notifier.SlashingNotifierArgs {
				return nil
			},
			expectedErr: process.ErrNilSlashingNotifierArgs,
		},
		{
			args: func() *notifier.SlashingNotifierArgs {
				args := generateSlashingNotifierArgs()
				args.KeyPairs = nil
				return args
			},
			expectedErr: process.ErrNilKeyPairs,
		},
		{
			args: func() *notifier.SlashingNotifierArgs {
				args := generateSlashingNotifierArgs()
				args.PubKeyConverter = nil
				return args
			},
			expectedErr: update.ErrNilPubKeyConverter,
		},
		{
			args: func() *notifier.SlashingNotifierArgs {
				args := generateSlashingNotifierArgs()
				args.Signer = nil
				return args
			},
			expectedErr: crypto.ErrNilSingleSigner,
		},
		{
			args: func() *notifier.SlashingNotifierArgs {
				args := generateSlashingNotifierArgs()
				args.AccountAdapter = nil
				return args
			},
			expectedErr: state.ErrNilAccountsAdapter,
		},
		{
			args: func() *notifier.SlashingNotifierArgs {
				args := generateSlashingNotifierArgs()
				args.Hasher = nil
				return args
			},
			expectedErr: process.ErrNilHasher,
		},
		{
			args: func() *notifier.SlashingNotifierArgs {
				args := generateSlashingNotifierArgs()
				args.Marshaller = nil
				return args
			},
			expectedErr: process.ErrNilMarshalizer,
		},
		{
			args: func() *notifier.SlashingNotifierArgs {
				args := generateSlashingNotifierArgs()
				args.ShardCoordinator = nil
				return args
			},
			expectedErr: process.ErrNilShardCoordinator,
		},
		{
			args: func() *notifier.SlashingNotifierArgs {
				return generateSlashingNotifierArgs()
			},
			expectedErr: nil,
		},
	}

	for _, currTest := range tests {
		_, err := notifier.NewSlashingNotifier(currTest.args())
		require.Equal(t, currTest.expectedErr, err)
	}
}

func TestSlashingNotifier_CreateShardSlashingTransaction_InvalidPubKey_ExpectError(t *testing.T) {
	args := generateSlashingNotifierArgs()
	errPubKey := errors.New("pub key error")

	pubKey := &cryptoMocks.PublicKeyStub{
		ToByteArrayStub: func() ([]byte, error) {
			return nil, errPubKey
		},
	}
	keyPair := notifier.KeyPair{
		PublicKey:  pubKey,
		PrivateKey: &mock.PrivateKeyMock{},
	}
	keyPairs := map[uint32]notifier.KeyPair{
		0: keyPair,
	}
	args.KeyPairs = keyPairs
	sn, _ := notifier.NewSlashingNotifier(args)
	tx, err := sn.CreateShardSlashingTransaction(&slashMocks.MultipleHeaderProposalProofStub{})
	require.Nil(t, tx)
	require.Equal(t, errPubKey, err)
}

func TestSlashingNotifier_CreateShardSlashingTransaction_InvalidAccount_ExpectError(t *testing.T) {
	args := generateSlashingNotifierArgs()
	errAcc := errors.New("accounts adapter error")
	args.AccountAdapter = &stateMock.AccountsStub{
		GetExistingAccountCalled: func([]byte) (vmcommon.AccountHandler, error) {
			return nil, errAcc
		},
	}

	sn, _ := notifier.NewSlashingNotifier(args)
	tx, err := sn.CreateShardSlashingTransaction(&slashMocks.MultipleHeaderProposalProofStub{})
	require.Nil(t, tx)
	require.Equal(t, errAcc, err)
}

func TestSlashingNotifier_CreateShardSlashingTransaction_InvalidMarshaller_ExpectError(t *testing.T) {
	args := generateSlashingNotifierArgs()
	errMarshaller := errors.New("marshaller error")
	args.Marshaller = &testscommon.MarshalizerStub{
		MarshalCalled: func(obj interface{}) ([]byte, error) {
			return nil, errMarshaller
		},
	}

	sn, _ := notifier.NewSlashingNotifier(args)
	tx, err := sn.CreateShardSlashingTransaction(&slashMocks.MultipleHeaderProposalProofStub{})
	require.Nil(t, tx)
	require.Equal(t, errMarshaller, err)
}

func TestSlashingNotifier_CreateShardSlashingTransaction_CannotGetDataForSigningBecauseOfInvalidMarshaller_ExpectError(t *testing.T) {
	args := generateSlashingNotifierArgs()
	errMarshaller := errors.New("marshaller error")
	flag := false
	args.Marshaller = &testscommon.MarshalizerStub{
		MarshalCalled: func(obj interface{}) ([]byte, error) {
			if flag {
				return nil, errMarshaller
			}
			flag = true
			return []byte("ok"), nil
		},
	}

	sn, _ := notifier.NewSlashingNotifier(args)
	tx, err := sn.CreateShardSlashingTransaction(&slashMocks.MultipleHeaderProposalProofStub{})
	require.Nil(t, tx)
	require.Equal(t, errMarshaller, err)
}

func TestSlashingNotifier_CreateShardSlashingTransaction_InvalidProofTxData_ExpectError(t *testing.T) {
	args := generateSlashingNotifierArgs()

	expectedErr := errors.New("invalid tx data extractor")
	proof := &slashMocks.MultipleHeaderProposalProofStub{
		GetProofTxDataCalled: func() (*coreSlash.ProofTxData, error) {
			return nil, expectedErr
		},
	}

	sn, _ := notifier.NewSlashingNotifier(args)
	tx, err := sn.CreateShardSlashingTransaction(proof)
	require.Nil(t, tx)
	require.Equal(t, expectedErr, err)
}

func TestSlashingNotifier_CreateShardSlashingTransaction_InvalidProofSignature_ExpectError(t *testing.T) {
	args := generateSlashingNotifierArgs()
	errSign := errors.New("signature error")
	args.Signer = &cryptoMocks.SignerStub{
		SignCalled: func(private crypto.PrivateKey, msg []byte) ([]byte, error) {
			return nil, errSign
		},
	}

	sn, _ := notifier.NewSlashingNotifier(args)
	tx, err := sn.CreateShardSlashingTransaction(&slashMocks.MultipleHeaderProposalProofStub{})
	require.Nil(t, tx)
	require.Equal(t, errSign, err)
}

func TestSlashingNotifier_CreateShardSlashingTransaction_InvalidTxSignature_ExpectError(t *testing.T) {
	args := generateSlashingNotifierArgs()
	errSign := errors.New("signature error")
	flag := false
	args.Signer = &cryptoMocks.SignerStub{
		SignCalled: func(private crypto.PrivateKey, msg []byte) ([]byte, error) {
			if flag {
				return nil, errSign
			}

			flag = true
			return []byte("signature"), nil
		},
	}

	sn, _ := notifier.NewSlashingNotifier(args)
	tx, err := sn.CreateShardSlashingTransaction(&slashMocks.MultipleHeaderProposalProofStub{})
	require.Nil(t, tx)
	require.Equal(t, errSign, err)
}

func TestSlashingNotifier_CreateShardSlashingTransaction_SelectKeyPairFromDifferentShard(t *testing.T) {
	shardID1 := uint32(1)
	shardID2 := uint32(2)

	pubKey1 := []byte("pubKey1")
	pubKey2 := []byte("pubKey2")

	flagPublicKey1 := atomic.Flag{}
	flagPublicKey2 := atomic.Flag{}

	privateKey1 := &mock.PrivateKeyMock{}
	publicKey1 := &mock.PublicKeyMock{
		ToByteArrayCalled: func() ([]byte, error) {
			flagPublicKey1.Set()
			return pubKey1, nil
		},
	}
	keyPair1 := notifier.KeyPair{
		PrivateKey: privateKey1,
		PublicKey:  publicKey1,
	}

	privateKey2 := &mock.PrivateKeyMock{}
	publicKey2 := &mock.PublicKeyMock{
		ToByteArrayCalled: func() ([]byte, error) {
			flagPublicKey2.Set()
			return pubKey2, nil
		},
	}
	keyPair2 := notifier.KeyPair{
		PrivateKey: privateKey2,
		PublicKey:  publicKey2,
	}

	keyPairs := map[uint32]notifier.KeyPair{
		shardID1: keyPair1,
		shardID2: keyPair2,
	}

	shardCoordinator := &testscommon.ShardsCoordinatorMock{
		SelfIDCalled: func() uint32 {
			return shardID1
		},
	}
	signer := &mockIntegration.SignerMock{
		SignStub: func(private crypto.PrivateKey, msg []byte) ([]byte, error) {
			require.True(t, privateKey2 == private)
			return []byte("signature"), nil
		},
	}

	expectedAccountAddr := []byte("accPubKey2")
	expectedNonce := uint64(123456)
	accountPubKey2 := &mockGenesis.BaseAccountMock{
		Nonce:             expectedNonce,
		AddressBytesField: expectedAccountAddr,
	}
	accountAdapter := &stateMock.AccountsStub{
		GetExistingAccountCalled: func(addressContainer []byte) (vmcommon.AccountHandler, error) {
			require.Equal(t, pubKey2, addressContainer)
			return accountPubKey2, nil
		},
	}

	args := generateSlashingNotifierArgs()
	args.Signer = signer
	args.KeyPairs = keyPairs
	args.AccountAdapter = accountAdapter
	args.ShardCoordinator = shardCoordinator

	sn, _ := notifier.NewSlashingNotifier(args)
	tx, err := sn.CreateShardSlashingTransaction(&slashMocks.MultipleHeaderProposalProofStub{})
	require.Nil(t, err)
	require.Equal(t, expectedNonce, tx.GetNonce())
	require.Equal(t, expectedAccountAddr, tx.GetSndAddr())
	require.False(t, flagPublicKey1.IsSet())
	require.True(t, flagPublicKey2.IsSet())
}

func TestSlashingNotifier_CreateShardSlashingTransaction_MultipleProposalProof(t *testing.T) {
	round := uint64(100000)
	shardID := uint32(2)

	args := generateSlashingNotifierArgs()
	args.Hasher = &testscommon.HasherStub{
		ComputeCalled: func(string) []byte {
			return []byte{byte('a'), byte('b'), byte('c'), byte('d')}
		},
	}
	sn, _ := notifier.NewSlashingNotifier(args)
	proof := &slashMocks.MultipleHeaderProposalProofStub{
		GetProofTxDataCalled: func() (*coreSlash.ProofTxData, error) {
			return &coreSlash.ProofTxData{
				Round:   round,
				ShardID: shardID,
				ProofID: coreSlash.MultipleProposalProofID,
			}, nil
		},
	}

	expectedData := []byte(fmt.Sprintf("%s@%s@%d@%d@%s@%s", notifier.BuiltInFunctionSlashCommitmentProof,
		[]byte{coreSlash.MultipleProposalProofID}, shardID, round, []byte{byte('c'), byte('d')}, []byte("signature")))

	expectedTx := &transaction.Transaction{
		Data:      expectedData,
		Nonce:     444,
		SndAddr:   []byte("address"),
		Value:     big.NewInt(notifier.CommitmentProofValue),
		GasPrice:  notifier.CommitmentProofGasPrice,
		GasLimit:  notifier.CommitmentProofGasLimit,
		Signature: []byte("signature"),
	}

	tx, _ := sn.CreateShardSlashingTransaction(proof)
	require.Equal(t, expectedTx, tx)
}

func TestSlashingNotifier_CreateShardSlashingTransaction_MultipleSignProof(t *testing.T) {
	round := uint64(100000)
	shardID := uint32(2)

	args := generateSlashingNotifierArgs()
	args.Hasher = &testscommon.HasherStub{
		ComputeCalled: func(string) []byte {
			return []byte{byte('a'), byte('b'), byte('c'), byte('d')}
		},
	}
	sn, _ := notifier.NewSlashingNotifier(args)
	proof := &slashMocks.MultipleHeaderSigningProofStub{
		GetProofTxDataCalled: func() (*coreSlash.ProofTxData, error) {
			return &coreSlash.ProofTxData{
				Round:   round,
				ShardID: shardID,
				ProofID: coreSlash.MultipleSigningProofID,
			}, nil
		},
	}

	expectedData := []byte(fmt.Sprintf("%s@%s@%d@%d@%s@%s", notifier.BuiltInFunctionSlashCommitmentProof,
		[]byte{coreSlash.MultipleSigningProofID}, shardID, round, []byte{byte('c'), byte('d')}, []byte("signature")))
	expectedTx := &transaction.Transaction{
		Data:      expectedData,
		Nonce:     444,
		SndAddr:   []byte("address"),
		Value:     big.NewInt(notifier.CommitmentProofValue),
		GasPrice:  notifier.CommitmentProofGasPrice,
		GasLimit:  notifier.CommitmentProofGasLimit,
		Signature: []byte("signature"),
	}

	tx, _ := sn.CreateShardSlashingTransaction(proof)
	require.Equal(t, expectedTx, tx)
}

func generateSlashingNotifierArgs() *notifier.SlashingNotifierArgs {
	accountHandler := &mockGenesis.BaseAccountMock{
		Nonce:             444,
		AddressBytesField: []byte("address"),
	}
	accountsAdapter := &stateMock.AccountsStub{
		GetExistingAccountCalled: func([]byte) (vmcommon.AccountHandler, error) {
			return accountHandler, nil
		},
	}
	marshaller := &testscommon.MarshalizerStub{
		MarshalCalled: func(obj interface{}) ([]byte, error) {
			return nil, nil
		},
	}
	shardID := uint32(0)
	shardCoordinatorMock := &testscommon.ShardsCoordinatorMock{
		SelfIDCalled: func() uint32 {
			return shardID
		},
	}
	keyPair := notifier.KeyPair{
		PrivateKey: &mock.PrivateKeyMock{},
		PublicKey:  &mock.PublicKeyMock{},
	}
	keyPairs := map[uint32]notifier.KeyPair{
		shardID: keyPair,
	}

	return &notifier.SlashingNotifierArgs{
		KeyPairs:         keyPairs,
		PubKeyConverter:  &testscommon.PubkeyConverterMock{},
		Signer:           &mockIntegration.SignerMock{},
		AccountAdapter:   accountsAdapter,
		Hasher:           &hashingMocks.HasherMock{},
		Marshaller:       marshaller,
		ShardCoordinator: shardCoordinatorMock,
	}
}
