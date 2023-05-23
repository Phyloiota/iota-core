//nolint:forcetypeassert,varnamelen,revive,exhaustruct // we don't care about these linters in test cases
package ledgerstate_test

import (
	"encoding/binary"
	"math"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/iotaledger/hive.go/kvstore"
	"github.com/iotaledger/hive.go/kvstore/mapdb"
	"github.com/iotaledger/hive.go/lo"
	"github.com/iotaledger/hive.go/serializer/v2/byteutils"
	"github.com/iotaledger/iota-core/pkg/protocol/engine/ledgerstate"
	"github.com/iotaledger/iota-core/pkg/protocol/engine/ledgerstate/tpkg"
	iotago "github.com/iotaledger/iota.go/v4"
)

func AssertOutputUnspentAndSpentTransitions(t *testing.T, output *ledgerstate.Output, spent *ledgerstate.Spent) {
	outputID := output.OutputID()
	manager := ledgerstate.New(mapdb.NewMapDB(), tpkg.API)

	require.NoError(t, manager.AddUnspentOutput(output))

	// Read Output from DB and compare
	readOutput, err := manager.ReadOutputByOutputID(outputID)
	require.NoError(t, err)
	tpkg.EqualOutput(t, output, readOutput)

	// Verify that it is unspent
	unspent, err := manager.IsOutputIDUnspentWithoutLocking(outputID)
	require.NoError(t, err)
	require.True(t, unspent)

	// Verify that all lookup keys exist in the database
	has, err := manager.KVStore().Has(output.UnspentLookupKey())
	require.NoError(t, err)
	require.True(t, has)

	// Spend it with a milestone
	require.NoError(t, manager.ApplyDiff(spent.SlotIndexSpent(), ledgerstate.Outputs{}, ledgerstate.Spents{spent}))

	// Read Spent from DB and compare
	readSpent, err := manager.ReadSpentForOutputIDWithoutLocking(outputID)
	require.NoError(t, err)
	tpkg.EqualSpent(t, spent, readSpent)

	// Verify that it is spent
	unspent, err = manager.IsOutputIDUnspentWithoutLocking(outputID)
	require.NoError(t, err)
	require.False(t, unspent)

	// Verify that no lookup keys exist in the database
	has, err = manager.KVStore().Has(output.UnspentLookupKey())
	require.NoError(t, err)
	require.False(t, has)

	// Rollback milestone
	require.NoError(t, manager.RollbackDiff(spent.SlotIndexSpent(), ledgerstate.Outputs{}, ledgerstate.Spents{spent}))

	// Verify that it is unspent
	unspent, err = manager.IsOutputIDUnspentWithoutLocking(outputID)
	require.NoError(t, err)
	require.True(t, unspent)

	// No Spent should be in the DB
	_, err = manager.ReadSpentForOutputIDWithoutLocking(outputID)
	require.ErrorIs(t, err, kvstore.ErrKeyNotFound)

	// Verify that all unspent keys exist in the database
	has, err = manager.KVStore().Has(output.UnspentLookupKey())
	require.NoError(t, err)
	require.True(t, has)
}

func CreateOutputAndAssertSerialization(t *testing.T, api iotago.API, blockID iotago.BlockID, indexBooked iotago.SlotIndex, timestampCreated time.Time, outputID iotago.OutputID, iotaOutput iotago.Output) *ledgerstate.Output {
	output := ledgerstate.CreateOutput(api, outputID, blockID, indexBooked, timestampCreated, iotaOutput)
	outputBytes, err := api.Encode(output.Output())
	require.NoError(t, err)

	require.Equal(t, byteutils.ConcatBytes([]byte{ledgerstate.StoreKeyPrefixOutput}, outputID[:]), output.KVStorableKey())

	value := output.KVStorableValue()
	require.Equal(t, blockID[:], value[:40])
	require.Equal(t, indexBooked, lo.PanicOnErr(iotago.SlotIndexFromBytes(value[40:48])))
	require.Equal(t, timestampCreated, time.Unix(0, int64(binary.LittleEndian.Uint64(value[48:56]))))
	require.Equal(t, outputBytes, value[56:])

	return output
}

func CreateSpentAndAssertSerialization(t *testing.T, output *ledgerstate.Output) *ledgerstate.Spent {
	transactionID := tpkg.RandTransactionID()

	indexSpent := iotago.SlotIndex(6788362)
	msTimestampSpent := tpkg.RandTimestamp()

	spent := ledgerstate.NewSpent(output, transactionID, msTimestampSpent, indexSpent)

	require.Equal(t, output, spent.Output())

	outputID := output.OutputID()
	require.Equal(t, byteutils.ConcatBytes([]byte{ledgerstate.StoreKeyPrefixOutputSpent}, outputID[:]), spent.KVStorableKey())

	value := spent.KVStorableValue()
	require.Equal(t, transactionID[:], value[:32])
	require.Equal(t, msTimestampSpent, time.Unix(0, int64(binary.LittleEndian.Uint64(value[32:40]))))
	require.Equal(t, indexSpent, lo.PanicOnErr(iotago.SlotIndexFromBytes(value[40:48])))

	return spent
}

func TestExtendedOutputOnEd25519WithoutSpendConstraintsSerialization(t *testing.T) {
	api := tpkg.API()
	outputID := tpkg.RandOutputID()
	blockID := tpkg.RandBlockID()
	address := tpkg.RandAddress(iotago.AddressEd25519).(*iotago.Ed25519Address)
	senderAddress := tpkg.RandAddress(iotago.AddressEd25519).(*iotago.Ed25519Address)
	tag := tpkg.RandBytes(23)
	amount := tpkg.RandAmount()
	index := tpkg.RandSlotIndex()
	timestampCreated := tpkg.RandTimestamp()

	iotaOutput := &iotago.BasicOutput{
		Amount: amount,
		Features: iotago.BasicOutputFeatures{
			&iotago.SenderFeature{
				Address: senderAddress,
			},
			&iotago.TagFeature{
				Tag: tag,
			},
		},
		Conditions: iotago.BasicOutputUnlockConditions{
			&iotago.AddressUnlockCondition{
				Address: address,
			},
		},
	}

	output := CreateOutputAndAssertSerialization(t, api, blockID, index, timestampCreated, outputID, iotaOutput)
	spent := CreateSpentAndAssertSerialization(t, output)

	require.ElementsMatch(t, byteutils.ConcatBytes([]byte{ledgerstate.StoreKeyPrefixOutputUnspent}, outputID[:]), output.UnspentLookupKey())
	AssertOutputUnspentAndSpentTransitions(t, output, spent)
}

func TestExtendedOutputOnEd25519WithSpendConstraintsSerialization(t *testing.T) {
	api := tpkg.API()
	outputID := tpkg.RandOutputID()
	blockID := tpkg.RandBlockID()
	address := tpkg.RandAddress(iotago.AddressEd25519).(*iotago.Ed25519Address)
	senderAddress := tpkg.RandAddress(iotago.AddressEd25519).(*iotago.Ed25519Address)
	amount := tpkg.RandAmount()
	index := tpkg.RandSlotIndex()
	timestampCreated := tpkg.RandTimestamp()

	iotaOutput := &iotago.BasicOutput{
		Amount: amount,
		Features: iotago.BasicOutputFeatures{
			&iotago.SenderFeature{
				Address: senderAddress,
			},
		},
		Conditions: iotago.BasicOutputUnlockConditions{
			&iotago.AddressUnlockCondition{
				Address: address,
			},
			&iotago.TimelockUnlockCondition{
				UnixTime: uint32(time.Now().Unix()),
			},
		},
	}

	output := CreateOutputAndAssertSerialization(t, api, blockID, index, timestampCreated, outputID, iotaOutput)
	spent := CreateSpentAndAssertSerialization(t, output)

	require.ElementsMatch(t, byteutils.ConcatBytes([]byte{ledgerstate.StoreKeyPrefixOutputUnspent}, outputID[:]), output.UnspentLookupKey())
	AssertOutputUnspentAndSpentTransitions(t, output, spent)
}

func TestNFTOutputSerialization(t *testing.T) {
	api := tpkg.API()
	outputID := tpkg.RandOutputID()
	blockID := tpkg.RandBlockID()
	address := tpkg.RandAddress(iotago.AddressEd25519).(*iotago.Ed25519Address)
	nftID := tpkg.RandNFTID()
	amount := tpkg.RandAmount()
	index := tpkg.RandSlotIndex()
	timestampCreated := tpkg.RandTimestamp()

	iotaOutput := &iotago.NFTOutput{
		Amount: amount,
		NFTID:  nftID,
		ImmutableFeatures: iotago.NFTOutputImmFeatures{
			&iotago.MetadataFeature{
				Data: tpkg.RandBytes(12),
			},
		},
		Conditions: iotago.NFTOutputUnlockConditions{
			&iotago.AddressUnlockCondition{
				Address: address,
			},
		},
	}

	output := CreateOutputAndAssertSerialization(t, api, blockID, index, timestampCreated, outputID, iotaOutput)
	spent := CreateSpentAndAssertSerialization(t, output)

	require.ElementsMatch(t, byteutils.ConcatBytes([]byte{ledgerstate.StoreKeyPrefixOutputUnspent}, outputID[:]), output.UnspentLookupKey())
	AssertOutputUnspentAndSpentTransitions(t, output, spent)
}

func TestNFTOutputWithSpendConstraintsSerialization(t *testing.T) {
	api := tpkg.API()
	outputID := tpkg.RandOutputID()
	blockID := tpkg.RandBlockID()
	address := tpkg.RandNFTID()
	issuerAddress := tpkg.RandAddress(iotago.AddressEd25519).(*iotago.Ed25519Address)
	nftID := tpkg.RandNFTID()
	amount := tpkg.RandAmount()
	index := tpkg.RandSlotIndex()
	timestampCreated := tpkg.RandTimestamp()

	iotaOutput := &iotago.NFTOutput{
		Amount: amount,
		NFTID:  nftID,
		ImmutableFeatures: iotago.NFTOutputImmFeatures{
			&iotago.MetadataFeature{
				Data: tpkg.RandBytes(12),
			},
			&iotago.IssuerFeature{
				Address: issuerAddress,
			},
		},
		Conditions: iotago.NFTOutputUnlockConditions{
			&iotago.AddressUnlockCondition{
				Address: address.ToAddress(),
			},
			&iotago.ExpirationUnlockCondition{
				UnixTime:      uint32(time.Now().Unix()),
				ReturnAddress: issuerAddress,
			},
		},
	}

	output := CreateOutputAndAssertSerialization(t, api, blockID, index, timestampCreated, outputID, iotaOutput)
	spent := CreateSpentAndAssertSerialization(t, output)

	require.ElementsMatch(t, byteutils.ConcatBytes([]byte{ledgerstate.StoreKeyPrefixOutputUnspent}, outputID[:]), output.UnspentLookupKey())
	AssertOutputUnspentAndSpentTransitions(t, output, spent)
}

func TestAccountOutputSerialization(t *testing.T) {
	api := tpkg.API()
	outputID := tpkg.RandOutputID()
	blockID := tpkg.RandBlockID()
	aliasID := tpkg.RandAccountID()
	stateController := tpkg.RandAccountID()
	governor := tpkg.RandAddress(iotago.AddressEd25519).(*iotago.Ed25519Address)
	issuer := tpkg.RandNFTID()
	sender := tpkg.RandAccountID()
	amount := tpkg.RandAmount()
	index := tpkg.RandSlotIndex()
	timestampCreated := tpkg.RandTimestamp()

	iotaOutput := &iotago.AccountOutput{
		Amount:    amount,
		AccountID: aliasID,
		Features: iotago.AccountOutputFeatures{
			&iotago.SenderFeature{
				Address: sender.ToAddress(),
			},
		},
		ImmutableFeatures: iotago.AccountOutputImmFeatures{
			&iotago.IssuerFeature{
				Address: issuer.ToAddress(),
			},
		},
		Conditions: iotago.AccountOutputUnlockConditions{
			&iotago.StateControllerAddressUnlockCondition{
				Address: stateController.ToAddress(),
			},
			&iotago.GovernorAddressUnlockCondition{
				Address: governor,
			},
		},
	}

	output := CreateOutputAndAssertSerialization(t, api, blockID, index, timestampCreated, outputID, iotaOutput)
	spent := CreateSpentAndAssertSerialization(t, output)

	require.ElementsMatch(t, byteutils.ConcatBytes([]byte{ledgerstate.StoreKeyPrefixOutputUnspent}, outputID[:]), output.UnspentLookupKey())
	AssertOutputUnspentAndSpentTransitions(t, output, spent)
}

func TestFoundryOutputSerialization(t *testing.T) {
	api := tpkg.API()
	outputID := tpkg.RandOutputID()
	blockID := tpkg.RandBlockID()
	aliasID := tpkg.RandAccountID()
	amount := tpkg.RandAmount()
	index := tpkg.RandSlotIndex()
	timestampCreated := tpkg.RandTimestamp()
	supply := new(big.Int).SetUint64(tpkg.RandAmount())

	iotaOutput := &iotago.FoundryOutput{
		Amount:       amount,
		SerialNumber: tpkg.RandUint32(math.MaxUint32),
		TokenScheme: &iotago.SimpleTokenScheme{
			MintedTokens:  supply,
			MeltedTokens:  new(big.Int).SetBytes([]byte{0}),
			MaximumSupply: supply,
		},
		Conditions: iotago.FoundryOutputUnlockConditions{
			&iotago.ImmutableAccountUnlockCondition{
				Address: aliasID.ToAddress().(*iotago.AccountAddress),
			},
		},
	}

	output := CreateOutputAndAssertSerialization(t, api, blockID, index, timestampCreated, outputID, iotaOutput)
	spent := CreateSpentAndAssertSerialization(t, output)

	require.ElementsMatch(t, byteutils.ConcatBytes([]byte{ledgerstate.StoreKeyPrefixOutputUnspent}, outputID[:]), output.UnspentLookupKey())
	AssertOutputUnspentAndSpentTransitions(t, output, spent)
}
