package itests

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"
	"time"

	"encoding/binary"
	"github.com/stretchr/testify/require"

	"github.com/filecoin-project/go-address"
	builtintypes "github.com/filecoin-project/go-state-types/builtin"
	"github.com/filecoin-project/go-state-types/manifest"

	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/chain/types/ethtypes"
	"github.com/filecoin-project/lotus/itests/kit"
)

//convert a simple byte array into input data which is a left padded 32 byte array
func inputDataFromArray(input []byte) []byte {
	inputData := make([]byte, 32)
	copy(inputData[32-len(input):], input[:])
	return inputData
}

//convert a "from" address into input data which is a left padded 32 byte array
func inputDataFromFrom(t *testing.T, ctx context.Context, client *kit.TestFullNode, from address.Address) []byte {
	fromId, err := client.StateLookupID(ctx, from, types.EmptyTSK)
	require.NoError(t, err)

	senderEthAddr, err := ethtypes.EthAddressFromFilecoinAddress(fromId)
	require.NoError(t, err)
	inputData := make([]byte, 32)
	copy(inputData[32-len(senderEthAddr):], senderEthAddr[:])
	return inputData
}

func setupFEVMTest(t *testing.T) (context.Context, context.CancelFunc, *kit.TestFullNode) {
	kit.QuietMiningLogs()
	blockTime := 100 * time.Millisecond
	client, _, ens := kit.EnsembleMinimal(t, kit.MockProofs(), kit.ThroughRPC())
	ens.InterconnectAll().BeginMining(blockTime)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	return ctx, cancel, client
}

func buildInputFromuint64(number uint64) []byte {
	// Convert the number to a binary uint64 array
	binaryNumber := make([]byte, 8)
	binary.BigEndian.PutUint64(binaryNumber, number)
	return inputDataFromArray(binaryNumber)
}

//math.Max is weird needs float
func max(x int, y int) int {
	max := x
	if y > max {
		max = y
	}
	return max
}

// TestFEVMRecursive does a basic fevm contract installation and invocation
func TestFEVMRecursive(t *testing.T) {
	callcounts := []uint64{0, 1}
	ctx, cancel, client := setupFEVMTest(t)
	defer cancel()
	filename := "contracts/Recursive.hex"
	fromAddr, idAddr := client.EVM().DeployContractFromFilename(ctx, filename)

	// Iterate over the numbers array
	for _, callcount := range callcounts {
		fmt.Println("mike - ", buildInputFromuint64(callcount))
		t.Run(fmt.Sprintf("TestFEVMRecursive%d", callcount), func(t *testing.T) {
			_, ret, err := client.EVM().InvokeContractByFuncName(ctx, fromAddr, idAddr, "recursiveCall(uint256)", buildInputFromuint64(callcount))
			if err != nil {
				fmt.Printf("error - %+v", err)
			}
			require.NoError(t, err)
			events := client.EVM().LoadEvents(ctx, *ret.Receipt.EventsRoot)
			//passing in 0 still means there's 1 event
			require.Equal(t, max(1, int(callcount)), len(events))
		})
	}
}

func TestFEVMRecursiveFail(t *testing.T) {
	callcounts := []uint64{2, 10, 1000, 100000}
	ctx, cancel, client := setupFEVMTest(t)
	defer cancel()
	filename := "contracts/Recursive.hex"
	fromAddr, idAddr := client.EVM().DeployContractFromFilename(ctx, filename)

	// Iterate over the numbers array
	for _, callcount := range callcounts {
		t.Run(fmt.Sprintf("TestFEVMRecursive%d", callcount), func(t *testing.T) {
			_, _, err := client.EVM().InvokeContractByFuncName(ctx, fromAddr, idAddr, "recursiveCall(uint256)", buildInputFromuint64(callcount))
			require.Error(t, err)
			require.True(t, strings.HasPrefix(err.Error(), "GasEstimateMessageGas"))
		})
	}
}
func TestFEVMRecursive1(t *testing.T) {
	callcount := 1
	ctx, cancel, client := setupFEVMTest(t)
	defer cancel()
	filename := "contracts/Recursive.hex"
	fromAddr, idAddr := client.EVM().DeployContractFromFilename(ctx, filename)
	_, ret, err := client.EVM().InvokeContractByFuncName(ctx, fromAddr, idAddr, "recursive1()", []byte{})
	require.NoError(t, err)
	events := client.EVM().LoadEvents(ctx, *ret.Receipt.EventsRoot)
	//passing in 0 still means there's 1 event
	require.Equal(t, max(1, int(callcount)), len(events))
}
func TestFEVMRecursive2(t *testing.T) {
	ctx, cancel, client := setupFEVMTest(t)
	defer cancel()
	filename := "contracts/Recursive.hex"
	fromAddr, idAddr := client.EVM().DeployContractFromFilename(ctx, filename)
	_, _, err := client.EVM().InvokeContractByFuncName(ctx, fromAddr, idAddr, "recursive2()", []byte{})
	require.Error(t, err)
	require.True(t, strings.HasPrefix(err.Error(), "GasEstimateMessageGas"))
}

// TestFEVMBasic does a basic fevm contract installation and invocation
func TestFEVMBasic(t *testing.T) {

	ctx, cancel, client := setupFEVMTest(t)
	defer cancel()

	filename := "contracts/SimpleCoin.hex"
	// install contract
	fromAddr, idAddr := client.EVM().DeployContractFromFilename(ctx, filename)

	// invoke the contract with owner
	{
		inputData := inputDataFromFrom(t, ctx, client, fromAddr)
		result, _, err := client.EVM().InvokeContractByFuncName(ctx, fromAddr, idAddr, "getBalance(address)", inputData)
		require.NoError(t, err)

		expectedResult, err := hex.DecodeString("0000000000000000000000000000000000000000000000000000000000002710")
		require.NoError(t, err)
		require.Equal(t, result, expectedResult)
	}

	// invoke the contract with non owner
	{
		inputData := inputDataFromFrom(t, ctx, client, fromAddr)
		inputData[31] += 1 // change the pub address to one that has 0 balance by modifying the last byte of the address
		result, _, err := client.EVM().InvokeContractByFuncName(ctx, fromAddr, idAddr, "getBalance(address)", inputData)
		require.NoError(t, err)

		expectedResult, err := hex.DecodeString("0000000000000000000000000000000000000000000000000000000000000000")
		require.NoError(t, err)
		require.Equal(t, result, expectedResult)
	}
}

// TestFEVMETH0 tests that the ETH0 actor is in genesis
func TestFEVMETH0(t *testing.T) {
	ctx, cancel, client := setupFEVMTest(t)
	defer cancel()

	eth0id, err := address.NewIDAddress(1001)
	require.NoError(t, err)

	client.AssertActorType(ctx, eth0id, manifest.EthAccountKey)

	act, err := client.StateGetActor(ctx, eth0id, types.EmptyTSK)
	require.NoError(t, err)

	eth0Addr, err := address.NewDelegatedAddress(builtintypes.EthereumAddressManagerActorID, make([]byte, 20))
	require.NoError(t, err)
	require.Equal(t, *act.Address, eth0Addr)
}

// TestFEVMDelegateCall deploys two contracts and makes a delegate call transaction
func TestFEVMDelegateCall(t *testing.T) {

	ctx, cancel, client := setupFEVMTest(t)
	defer cancel()

	//install contract Actor
	filenameActor := "contracts/DelegatecallActor.hex"
	fromAddr, actorAddr := client.EVM().DeployContractFromFilename(ctx, filenameActor)
	//install contract Storage
	filenameStorage := "contracts/DelegatecallStorage.hex"
	fromAddrStorage, storageAddr := client.EVM().DeployContractFromFilename(ctx, filenameStorage)
	require.Equal(t, fromAddr, fromAddrStorage)

	//call Contract Storage which makes a delegatecall to contract Actor
	//this contract call sets the "counter" variable to 7, from default value 0

	inputDataContract := inputDataFromFrom(t, ctx, client, actorAddr)
	inputDataValue := inputDataFromArray([]byte{7})
	inputData := append(inputDataContract, inputDataValue...)

	//verify that the returned value of the call to setvars is 7
	result, _, err := client.EVM().InvokeContractByFuncName(ctx, fromAddr, storageAddr, "setVars(address,uint256)", inputData)
	require.NoError(t, err)
	expectedResult, err := hex.DecodeString("0000000000000000000000000000000000000000000000000000000000000007")
	require.NoError(t, err)
	require.Equal(t, result, expectedResult)

	//test the value is 7 via calling the getter
	result, _, err = client.EVM().InvokeContractByFuncName(ctx, fromAddr, storageAddr, "getCounter()", []byte{})
	require.NoError(t, err)
	require.Equal(t, result, expectedResult)

	//test the value is 0 via calling the getter on the Actor contract
	result, _, err = client.EVM().InvokeContractByFuncName(ctx, fromAddr, actorAddr, "getCounter()", []byte{})
	require.NoError(t, err)
	expectedResultActor, err := hex.DecodeString("0000000000000000000000000000000000000000000000000000000000000000")
	require.NoError(t, err)
	require.Equal(t, result, expectedResultActor)
}
