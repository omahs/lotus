package itests

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/chain/types/ethtypes"
	"github.com/filecoin-project/lotus/itests/kit"
)

func TestEthGetBalanceExistingF4address(t *testing.T) {
	blockTime := 100 * time.Millisecond
	client, _, ens := kit.EnsembleMinimal(t, kit.MockProofs(), kit.ThroughRPC())
	ens.InterconnectAll().BeginMining(blockTime)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	_, ethAddr, deployer := client.EVM().NewAccount()

	fundAmount := types.FromFil(10)
	// send some funds to the f410 address
	kit.SendFunds(ctx, t, client, deployer, fundAmount)

	balance, err := client.EthGetBalance(ctx, ethAddr, "latest")
	require.NoError(t, err)
	require.Equal(t, balance, ethtypes.EthBigInt{Int: fundAmount.Int})
}

func TestEthGetBalanceNonExistentF4address(t *testing.T) {
	blockTime := 100 * time.Millisecond
	client, _, ens := kit.EnsembleMinimal(t, kit.MockProofs(), kit.ThroughRPC())
	ens.InterconnectAll().BeginMining(blockTime)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	_, ethAddr, _ := client.EVM().NewAccount()

	balance, err := client.EthGetBalance(ctx, ethAddr, "latest")
	require.NoError(t, err)
	require.Equal(t, balance, ethtypes.EthBigIntZero)
}

func TestEthGetBalanceExistentIDMaskedAddr(t *testing.T) {
	blockTime := 100 * time.Millisecond
	client, _, ens := kit.EnsembleMinimal(t, kit.MockProofs(), kit.ThroughRPC())
	ens.InterconnectAll().BeginMining(blockTime)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	faddr, err := client.WalletDefaultAddress(ctx)
	require.NoError(t, err)
	fid, err := client.StateLookupID(ctx, faddr, types.EmptyTSK)
	require.NoError(t, err)

	ethAddr, err := ethtypes.EthAddressFromFilecoinAddress(fid)
	require.NoError(t, err)

	balance, err := client.WalletBalance(ctx, fid)
	require.NoError(t, err)

	ebal, err := client.EthGetBalance(ctx, ethAddr, "latest")
	require.NoError(t, err)
	require.Equal(t, ebal, ethtypes.EthBigInt{Int: balance.Int})
}