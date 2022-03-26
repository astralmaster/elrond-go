package trieIterators

import (
	"fmt"
	"math/big"

	"github.com/ElrondNetwork/elrond-go-core/data"
	"github.com/ElrondNetwork/elrond-go/epochStart"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/state"
	"github.com/ElrondNetwork/elrond-go/vm"
	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
)

type stakingValidatorInfo struct {
	totalStakedValue *big.Int
	topUpValue       *big.Int
	unstakedValue    *big.Int
}

type commonStakingProcessor struct {
	queryService process.SCQueryService
	blockChain   data.ChainHandler
	accounts     *AccountsWrapper
}

func (csp *commonStakingProcessor) getValidatorInfoFromSC(validatorAddress []byte) (*stakingValidatorInfo, error) {
	scQuery := &process.SCQuery{
		ScAddress:  vm.ValidatorSCAddress,
		FuncName:   "getTotalStakedTopUpStakedBlsKeys",
		CallerAddr: vm.ValidatorSCAddress,
		CallValue:  big.NewInt(0),
		Arguments:  [][]byte{validatorAddress},
	}

	vmOutput, err := csp.queryService.ExecuteQuery(scQuery)
	if err != nil {
		return nil, err
	}
	if vmOutput.ReturnCode != vmcommon.Ok {
		return nil, fmt.Errorf("%w, return code: %v, message: %s", epochStart.ErrExecutingSystemScCode, vmOutput.ReturnCode, vmOutput.ReturnMessage)
	}

	if len(vmOutput.ReturnData) < 3 {
		return nil, fmt.Errorf("%w, getTotalStakedTopUpStakedBlsKeys function should have at least three values", epochStart.ErrExecutingSystemScCode)
	}

	info := &stakingValidatorInfo{}

	info.topUpValue = big.NewInt(0).SetBytes(vmOutput.ReturnData[0])
	info.totalStakedValue = big.NewInt(0).SetBytes(vmOutput.ReturnData[1])
	unstakedValue, err := csp.getValidatorUnstakedValue(validatorAddress)
	if err != nil {
		return nil, err
	}
	info.unstakedValue = unstakedValue

	return info, nil
}

func (csp *commonStakingProcessor) getValidatorUnstakedValue(validatorAddress []byte) (*big.Int, error) {
	scQuery := &process.SCQuery{
		ScAddress:  vm.ValidatorSCAddress,
		FuncName:   "getUnStakedTokensList",
		CallerAddr: vm.ValidatorSCAddress,
		CallValue:  big.NewInt(0),
		Arguments:  [][]byte{validatorAddress},
	}

	vmOutput, err := csp.queryService.ExecuteQuery(scQuery)
	if err != nil {
		return nil, err
	}
	if vmOutput.ReturnCode != vmcommon.Ok {
		return nil, fmt.Errorf("%w, return code: %v, message: %s", epochStart.ErrExecutingSystemScCode, vmOutput.ReturnCode, vmOutput.ReturnMessage)
	}

	if len(vmOutput.ReturnData) % 2 != 0 {
		return nil, fmt.Errorf("%w, getUnStakedTokensList function should have an even number of values", epochStart.ErrExecutingSystemScCode)
	}

	unstakedValue := big.NewInt(0)
	for i := 0; i < len(vmOutput.ReturnData); i+=2 {
		unstakedValue.Add(unstakedValue, big.NewInt(0).SetBytes(vmOutput.ReturnData[i]))
	}

	return unstakedValue, nil
}

func (csp *commonStakingProcessor) getAccount(scAddress []byte) (state.UserAccountHandler, error) {
	accountHandler, err := csp.accounts.GetExistingAccount(scAddress)
	if err != nil {
		return nil, err
	}

	account, ok := accountHandler.(state.UserAccountHandler)
	if !ok {
		return nil, ErrCannotCastAccountHandlerToUserAccount
	}

	return account, nil
}
