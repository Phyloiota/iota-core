//nolint:forcetypeassert,varnamelen,revive,exhaustruct // we don't care about these linters in test cases
package tpkg

import (
	"bytes"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/iotaledger/iota-core/pkg/protocol/engine/ledgerstate"
	iotago "github.com/iotaledger/iota.go/v4"
)

func EqualOutput(t *testing.T, expected *ledgerstate.Output, actual *ledgerstate.Output) {
	require.Equal(t, expected.OutputID(), actual.OutputID())
	require.Equal(t, expected.BlockID(), actual.BlockID())
	require.Equal(t, expected.SlotIndexBooked(), actual.SlotIndexBooked())
	require.Equal(t, expected.TimestampCreated(), actual.TimestampCreated())
	require.Equal(t, expected.OutputType(), actual.OutputType())

	var expectedIdent iotago.Address
	switch output := expected.Output().(type) {
	case iotago.TransIndepIdentOutput:
		expectedIdent = output.Ident()
	case iotago.TransDepIdentOutput:
		expectedIdent = output.Chain().ToAddress()
	default:
		require.Fail(t, "unsupported output type")
	}

	var actualIdent iotago.Address
	switch output := actual.Output().(type) {
	case iotago.TransIndepIdentOutput:
		actualIdent = output.Ident()
	case iotago.TransDepIdentOutput:
		actualIdent = output.Chain().ToAddress()
	default:
		require.Fail(t, "unsupported output type")
	}

	require.NotNil(t, expectedIdent)
	require.NotNil(t, actualIdent)
	require.True(t, expectedIdent.Equal(actualIdent))
	require.Equal(t, expected.Deposit(), actual.Deposit())
	require.EqualValues(t, expected.Output(), actual.Output())
}

func EqualSpent(t *testing.T, expected *ledgerstate.Spent, actual *ledgerstate.Spent) {
	require.Equal(t, expected.OutputID(), actual.OutputID())
	require.Equal(t, expected.TransactionIDSpent(), actual.TransactionIDSpent())
	require.Equal(t, expected.SlotIndexSpent(), actual.SlotIndexSpent())
	require.Equal(t, expected.TimestampSpent(), actual.TimestampSpent())
	EqualOutput(t, expected.Output(), actual.Output())
}

func EqualOutputs(t *testing.T, expected ledgerstate.Outputs, actual ledgerstate.Outputs) {
	require.Equal(t, len(expected), len(actual))

	// Sort Outputs by output ID.
	sort.Slice(expected, func(i, j int) bool {
		iOutputID := expected[i].OutputID()
		jOutputID := expected[j].OutputID()

		return bytes.Compare(iOutputID[:], jOutputID[:]) == -1
	})
	sort.Slice(actual, func(i, j int) bool {
		iOutputID := actual[i].OutputID()
		jOutputID := actual[j].OutputID()

		return bytes.Compare(iOutputID[:], jOutputID[:]) == -1
	})

	for i := 0; i < len(expected); i++ {
		EqualOutput(t, expected[i], actual[i])
	}
}

func EqualSpents(t *testing.T, expected ledgerstate.Spents, actual ledgerstate.Spents) {
	require.Equal(t, len(expected), len(actual))

	// Sort Spents by output ID.
	sort.Slice(expected, func(i, j int) bool {
		iOutputID := expected[i].OutputID()
		jOutputID := expected[j].OutputID()

		return bytes.Compare(iOutputID[:], jOutputID[:]) == -1
	})
	sort.Slice(actual, func(i, j int) bool {
		iOutputID := actual[i].OutputID()
		jOutputID := actual[j].OutputID()

		return bytes.Compare(iOutputID[:], jOutputID[:]) == -1
	})

	for i := 0; i < len(expected); i++ {
		EqualSpent(t, expected[i], actual[i])
	}
}
