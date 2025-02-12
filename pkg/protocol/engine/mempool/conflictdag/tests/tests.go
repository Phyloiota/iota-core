package tests

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/iotaledger/iota-core/pkg/core/acceptance"
)

func TestAll(t *testing.T, frameworkProvider func(*testing.T) *Framework) {
	for testName, testCase := range map[string]func(*testing.T, *Framework){
		"CreateConflict":                        CreateConflict,
		"TestExistingConflictJoinsConflictSets": TestExistingConflictJoinsConflictSets,
		"UpdateConflictParents":                 UpdateConflictParents,
		"LikedInstead":                          LikedInstead,
		"CreateConflictWithoutMembers":          CreateConflictWithoutMembers,
		"ConflictAcceptance":                    ConflictAcceptance,
		"CastVotes":                             CastVotes,
		"CastVotes_VotePower":                   CastVotesVotePower,
		"CastVotesAcceptance":                   CastVotesAcceptance,
	} {
		t.Run(testName, func(t *testing.T) { testCase(t, frameworkProvider(t)) })
	}
}

func TestExistingConflictJoinsConflictSets(t *testing.T, tf *Framework) {
	require.NoError(tf.test, tf.CreateOrUpdateConflict("conflict1", []string{"resource1"}, acceptance.Pending))
	require.NoError(t, tf.CreateOrUpdateConflict("conflict2", []string{"resource1"}, acceptance.Rejected))

	// require.ErrorIs(t, tf.JoinConflictSets("conflict2", "resource2"), conflictdag.ErrEntityEvicted, "modifying rejected conflicts should fail with ErrEntityEvicted")

	require.NoError(t, tf.CreateOrUpdateConflict("conflict3", []string{"resource2"}, acceptance.Pending))
	require.NoError(t, tf.CreateOrUpdateConflict("conflict1", []string{"resource2"}))
	tf.Assert.ConflictSetMembers("resource2", "conflict1", "conflict3")

	require.NoError(t, tf.CreateOrUpdateConflict("conflict2", []string{"resource2"}))
	tf.Assert.ConflictSetMembers("resource2", "conflict1", "conflict2", "conflict3")

	tf.Assert.LikedInstead([]string{"conflict3"}, "conflict1")
}

func UpdateConflictParents(t *testing.T, tf *Framework) {
	require.NoError(t, tf.CreateOrUpdateConflict("conflict1", []string{"resource1"}))
	require.NoError(t, tf.CreateOrUpdateConflict("conflict2", []string{"resource2"}))

	require.NoError(t, tf.CreateOrUpdateConflict("conflict3", []string{"resource1", "resource2"}))
	require.NoError(t, tf.UpdateConflictParents("conflict3", []string{"conflict1", "conflict2"}, []string{}))
	tf.Assert.Children("conflict1", "conflict3")
	tf.Assert.Parents("conflict3", "conflict1", "conflict2")

	require.NoError(t, tf.CreateOrUpdateConflict("conflict2.5", []string{"conflict2.5"}))
	require.NoError(t, tf.UpdateConflictParents("conflict2.5", []string{"conflict1", "conflict2"}, []string{}))
	tf.Assert.Children("conflict1", "conflict2.5", "conflict3")
	tf.Assert.Children("conflict2", "conflict2.5", "conflict3")
	tf.Assert.Parents("conflict2.5", "conflict1", "conflict2")

	require.NoError(t, tf.UpdateConflictParents("conflict3", []string{"conflict2.5"}, []string{"conflict1", "conflict2"}))

	tf.Assert.Children("conflict1", "conflict2.5")
	tf.Assert.Children("conflict2", "conflict2.5")
	tf.Assert.Children("conflict2.5", "conflict3")
	tf.Assert.Parents("conflict3", "conflict2.5")
	tf.Assert.Parents("conflict2.5", "conflict1", "conflict2")
}

func CreateConflict(t *testing.T, tf *Framework) {
	require.NoError(t, tf.CreateOrUpdateConflict("conflict1", []string{"resource1"}))
	require.NoError(t, tf.CreateOrUpdateConflict("conflict2", []string{"resource1"}))
	tf.Assert.ConflictSetMembers("resource1", "conflict1", "conflict2")

	require.NoError(t, tf.CreateOrUpdateConflict("conflict3", []string{"resource2"}))
	require.NoError(t, tf.UpdateConflictParents("conflict3", []string{"conflict1"}, []string{}))

	require.NoError(t, tf.CreateOrUpdateConflict("conflict4", []string{"resource2"}))
	require.NoError(t, tf.UpdateConflictParents("conflict4", []string{"conflict1"}, []string{}))

	tf.Assert.ConflictSetMembers("resource2", "conflict3", "conflict4")
	tf.Assert.Children("conflict1", "conflict3", "conflict4")
	tf.Assert.Parents("conflict3", "conflict1")
	tf.Assert.Parents("conflict4", "conflict1")
}

func CreateConflictWithoutMembers(t *testing.T, tf *Framework) {
	tf.Accounts.CreateID("nodeID1", 10)
	tf.Accounts.CreateID("nodeID2", 10)
	tf.Accounts.CreateID("nodeID3", 10)
	tf.Accounts.CreateID("nodeID4", 0)

	// Non-conflicting conflicts
	{
		require.NoError(t, tf.CreateOrUpdateConflict("conflict1", []string{"resource1"}))
		require.NoError(t, tf.CreateOrUpdateConflict("conflict2", []string{"resource2"}))

		tf.Assert.ConflictSetMembers("resource1", "conflict1")
		tf.Assert.ConflictSetMembers("resource2", "conflict2")

		tf.Assert.LikedInstead([]string{"conflict1"})
		tf.Assert.LikedInstead([]string{"conflict2"})

		require.NoError(t, tf.CastVotes("nodeID1", 1, "conflict1"))
		require.NoError(t, tf.CastVotes("nodeID2", 1, "conflict1"))
		require.NoError(t, tf.CastVotes("nodeID3", 1, "conflict1"))

		tf.Assert.LikedInstead([]string{"conflict1"})
		tf.Assert.Accepted("conflict1")
	}

	// Regular conflict
	{
		require.NoError(t, tf.CreateOrUpdateConflict("conflict3", []string{"resource3"}))
		require.NoError(t, tf.CreateOrUpdateConflict("conflict4", []string{"resource3"}))

		tf.Assert.ConflictSetMembers("resource3", "conflict3", "conflict4")

		require.NoError(t, tf.CastVotes("nodeID3", 1, "conflict3"))

		tf.Assert.LikedInstead([]string{"conflict3"})
		tf.Assert.LikedInstead([]string{"conflict4"}, "conflict3")
	}

	tf.Assert.LikedInstead([]string{"conflict1", "conflict4"}, "conflict3")
}

func LikedInstead(t *testing.T, tf *Framework) {
	tf.Accounts.CreateID("zero-weight")

	require.NoError(t, tf.CreateOrUpdateConflict("conflict1", []string{"resource1"}))
	require.NoError(t, tf.CastVotes("zero-weight", 1, "conflict1"))
	require.NoError(t, tf.CreateOrUpdateConflict("conflict2", []string{"resource1"}))
	tf.Assert.ConflictSetMembers("resource1", "conflict1", "conflict2")
	tf.Assert.LikedInstead([]string{"conflict1", "conflict2"}, "conflict1")

	require.NoError(t, tf.CreateOrUpdateConflict("conflict3", []string{"resource2"}))
	require.NoError(t, tf.UpdateConflictParents("conflict3", []string{"conflict1"}, []string{}))

	require.NoError(t, tf.CreateOrUpdateConflict("conflict4", []string{"resource2"}))
	require.NoError(t, tf.UpdateConflictParents("conflict4", []string{"conflict1"}, []string{}))

	require.NoError(t, tf.CastVotes("zero-weight", 1, "conflict4"))
	tf.Assert.LikedInstead([]string{"conflict1", "conflict2", "conflict3", "conflict4"}, "conflict1", "conflict4")
}

func ConflictAcceptance(t *testing.T, tf *Framework) {
	tf.Accounts.CreateID("nodeID1", 10)
	tf.Accounts.CreateID("nodeID2", 10)
	tf.Accounts.CreateID("nodeID3", 10)
	tf.Accounts.CreateID("nodeID4", 10)

	require.NoError(t, tf.CreateOrUpdateConflict("conflict1", []string{"resource1"}))
	require.NoError(t, tf.CreateOrUpdateConflict("conflict2", []string{"resource1"}))
	tf.Assert.ConflictSetMembers("resource1", "conflict1", "conflict2")
	tf.Assert.ConflictSets("conflict1", "resource1")
	tf.Assert.ConflictSets("conflict2", "resource1")

	require.NoError(t, tf.CreateOrUpdateConflict("conflict3", []string{"resource2"}))
	require.NoError(t, tf.UpdateConflictParents("conflict3", []string{"conflict1"}, []string{}))

	require.NoError(t, tf.CreateOrUpdateConflict("conflict4", []string{"resource2"}))
	require.NoError(t, tf.UpdateConflictParents("conflict4", []string{"conflict1"}, []string{}))

	tf.Assert.ConflictSetMembers("resource2", "conflict3", "conflict4")
	tf.Assert.Children("conflict1", "conflict3", "conflict4")
	tf.Assert.Parents("conflict3", "conflict1")
	tf.Assert.Parents("conflict4", "conflict1")

	require.NoError(t, tf.CastVotes("nodeID1", 1, "conflict4"))
	require.NoError(t, tf.CastVotes("nodeID2", 1, "conflict4"))
	require.NoError(t, tf.CastVotes("nodeID3", 1, "conflict4"))

	tf.Assert.LikedInstead([]string{"conflict1"})
	tf.Assert.LikedInstead([]string{"conflict2"}, "conflict1")
	tf.Assert.LikedInstead([]string{"conflict3"}, "conflict4")
	tf.Assert.LikedInstead([]string{"conflict4"})

	tf.Assert.Accepted("conflict1", "conflict4")
}

func CastVotes(t *testing.T, tf *Framework) {
	tf.Accounts.CreateID("nodeID1", 10)
	tf.Accounts.CreateID("nodeID2", 10)
	tf.Accounts.CreateID("nodeID3", 10)
	tf.Accounts.CreateID("nodeID4", 10)

	require.NoError(t, tf.CreateOrUpdateConflict("conflict1", []string{"resource1"}))
	require.NoError(t, tf.CreateOrUpdateConflict("conflict2", []string{"resource1"}))
	tf.Assert.ConflictSetMembers("resource1", "conflict1", "conflict2")
	tf.Assert.ConflictSets("conflict1", "resource1")
	tf.Assert.ConflictSets("conflict2", "resource1")

	require.NoError(t, tf.CreateOrUpdateConflict("conflict3", []string{"resource2"}))
	require.NoError(t, tf.UpdateConflictParents("conflict3", []string{"conflict1"}, []string{}))

	require.NoError(t, tf.CreateOrUpdateConflict("conflict4", []string{"resource2"}))
	require.NoError(t, tf.UpdateConflictParents("conflict4", []string{"conflict1"}, []string{}))

	tf.Assert.ConflictSetMembers("resource2", "conflict3", "conflict4")
	tf.Assert.Children("conflict1", "conflict3", "conflict4")
	tf.Assert.Parents("conflict3", "conflict1")
	tf.Assert.Parents("conflict4", "conflict1")

	require.NoError(t, tf.CastVotes("nodeID1", 1, "conflict2"))
	require.NoError(t, tf.CastVotes("nodeID2", 1, "conflict2"))
	require.NoError(t, tf.CastVotes("nodeID3", 1, "conflict2"))
	tf.Assert.LikedInstead([]string{"conflict1"}, "conflict2")

	tf.Assert.Accepted("conflict2")
	tf.Assert.Rejected("conflict1")
	tf.Assert.Rejected("conflict3")
	tf.Assert.Rejected("conflict4")

	require.Error(t, tf.CastVotes("nodeID3", 1, "conflict1", "conflict2"))
}

func CastVotesVotePower(t *testing.T, tf *Framework) {
	tf.Accounts.CreateID("nodeID1", 10)
	tf.Accounts.CreateID("nodeID2", 10)
	tf.Accounts.CreateID("nodeID3", 10)
	tf.Accounts.CreateID("nodeID4", 0)

	require.NoError(t, tf.CreateOrUpdateConflict("conflict1", []string{"resource1"}))
	require.NoError(t, tf.CreateOrUpdateConflict("conflict2", []string{"resource1"}))
	tf.Assert.ConflictSetMembers("resource1", "conflict1", "conflict2")
	tf.Assert.ConflictSets("conflict1", "resource1")
	tf.Assert.ConflictSets("conflict2", "resource1")

	// create nested conflicts
	require.NoError(t, tf.CreateOrUpdateConflict("conflict3", []string{"resource2"}))
	require.NoError(t, tf.UpdateConflictParents("conflict3", []string{"conflict1"}, []string{}))

	require.NoError(t, tf.CreateOrUpdateConflict("conflict4", []string{"resource2"}))
	require.NoError(t, tf.UpdateConflictParents("conflict4", []string{"conflict1"}, []string{}))

	tf.Assert.ConflictSetMembers("resource2", "conflict3", "conflict4")
	tf.Assert.Children("conflict1", "conflict3", "conflict4")
	tf.Assert.Parents("conflict3", "conflict1")
	tf.Assert.Parents("conflict4", "conflict1")

	// casting a vote from non-relevant validator before any relevant validators increases validator weight
	// require.NoError(t, tf.CastVotes("nodeID4", 2, "conflict3"))
	// tf.Assert.LikedInstead([]string{"conflict1"})
	// tf.Assert.LikedInstead([]string{"conflict2"}, "conflict1")
	// tf.Assert.LikedInstead([]string{"conflict3"})
	// tf.Assert.LikedInstead([]string{"conflict4"}, "conflict3")

	// casting a vote from non-relevant validator before any relevant validators increases validator weight
	// require.NoError(t, tf.CastVotes("nodeID4", 2, "conflict2"))
	// require.NoError(t, tf.CastVotes("nodeID4", 2, "conflict2"))
	// tf.Assert.LikedInstead([]string{"conflict1"}, "conflict2")
	// tf.Assert.LikedInstead([]string{"conflict2"})
	// tf.Assert.LikedInstead([]string{"conflict3"}, "conflict2")
	// tf.Assert.LikedInstead([]string{"conflict4"}, "conflict2")

	// casting a vote from a validator updates the validator weight
	require.NoError(t, tf.CastVotes("nodeID1", 2, "conflict4"))
	tf.Assert.LikedInstead([]string{"conflict1"})
	tf.Assert.LikedInstead([]string{"conflict2"}, "conflict1")
	tf.Assert.LikedInstead([]string{"conflict3"}, "conflict4")
	tf.Assert.LikedInstead([]string{"conflict4"})

	// casting a vote from non-relevant validator after processing a vote from relevant validator doesn't change weights
	require.NoError(t, tf.CastVotes("nodeID4", 2, "conflict2"))
	require.NoError(t, tf.CastVotes("nodeID4", 2, "conflict2"))
	tf.Assert.LikedInstead([]string{"conflict1"})
	tf.Assert.LikedInstead([]string{"conflict2"}, "conflict1")
	tf.Assert.LikedInstead([]string{"conflict3"}, "conflict4")
	tf.Assert.LikedInstead([]string{"conflict4"})
	tf.Assert.ValidatorWeight("conflict1", 10)
	tf.Assert.ValidatorWeight("conflict2", 0)
	tf.Assert.ValidatorWeight("conflict3", 0)
	tf.Assert.ValidatorWeight("conflict4", 10)

	// casting vote with lower vote power doesn't change the weights of conflicts
	require.NoError(t, tf.CastVotes("nodeID1", 1), "conflict3")
	tf.Assert.LikedInstead([]string{"conflict1"})
	tf.Assert.LikedInstead([]string{"conflict2"}, "conflict1")
	tf.Assert.LikedInstead([]string{"conflict3"}, "conflict4")
	tf.Assert.LikedInstead([]string{"conflict4"})
	tf.Assert.ValidatorWeight("conflict1", 10)
	tf.Assert.ValidatorWeight("conflict2", 0)
	tf.Assert.ValidatorWeight("conflict3", 0)
	tf.Assert.ValidatorWeight("conflict4", 10)

	// casting vote with higher vote power changes the weights of conflicts
	require.NoError(t, tf.CastVotes("nodeID1", 3, "conflict3"))
	tf.Assert.LikedInstead([]string{"conflict1"})
	tf.Assert.LikedInstead([]string{"conflict2"}, "conflict1")
	tf.Assert.LikedInstead([]string{"conflict3"})
	tf.Assert.LikedInstead([]string{"conflict4"}, "conflict3")
	tf.Assert.ValidatorWeight("conflict1", 10)
	tf.Assert.ValidatorWeight("conflict2", 0)
	tf.Assert.ValidatorWeight("conflict3", 10)
	tf.Assert.ValidatorWeight("conflict4", 0)
}

func CastVotesAcceptance(t *testing.T, tf *Framework) {
	tf.Accounts.CreateID("nodeID1", 10)
	tf.Accounts.CreateID("nodeID2", 10)
	tf.Accounts.CreateID("nodeID3", 10)
	tf.Accounts.CreateID("nodeID4", 10)

	require.NoError(t, tf.CreateOrUpdateConflict("conflict1", []string{"resource1"}))
	require.NoError(t, tf.CreateOrUpdateConflict("conflict2", []string{"resource1"}))
	tf.Assert.ConflictSetMembers("resource1", "conflict1", "conflict2")
	tf.Assert.ConflictSets("conflict1", "resource1")
	tf.Assert.ConflictSets("conflict2", "resource1")

	require.NoError(t, tf.CreateOrUpdateConflict("conflict3", []string{"resource2"}))
	require.NoError(t, tf.UpdateConflictParents("conflict3", []string{"conflict1"}, []string{}))

	require.NoError(t, tf.CreateOrUpdateConflict("conflict4", []string{"resource2"}))
	require.NoError(t, tf.UpdateConflictParents("conflict4", []string{"conflict1"}, []string{}))

	tf.Assert.ConflictSetMembers("resource2", "conflict3", "conflict4")
	tf.Assert.Children("conflict1", "conflict3", "conflict4")
	tf.Assert.Parents("conflict3", "conflict1")
	tf.Assert.Parents("conflict4", "conflict1")

	require.NoError(t, tf.CastVotes("nodeID1", 1, "conflict3"))
	require.NoError(t, tf.CastVotes("nodeID2", 1, "conflict3"))
	require.NoError(t, tf.CastVotes("nodeID3", 1, "conflict3"))
	tf.Assert.LikedInstead([]string{"conflict1"})
	tf.Assert.Accepted("conflict1")
	tf.Assert.Rejected("conflict2")
	tf.Assert.Accepted("conflict3")
	tf.Assert.Rejected("conflict4")
}
