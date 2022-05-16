package mselection

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/gohornet/hornet/pkg/model/hornet"
	"github.com/gohornet/inx-coordinator/pkg/utils"
	"github.com/iotaledger/inx/go"
	iotago "github.com/iotaledger/iota.go/v3"
)

const (
	CfgCoordinatorTipselectMinHeaviestBranchUnreferencedBlocksThreshold = 20
	CfgCoordinatorTipselectMaxHeaviestBranchTipsPerCheckpoint           = 10
	CfgCoordinatorTipselectRandomTipsPerCheckpoint                      = 3
	CfgCoordinatorTipselectHeaviestBranchSelectionTimeoutMilliseconds   = 100

	numTestBlocks      = 32 * 100
	numBenchmarkBlocks = 5000
)

func init() {
	rand.Seed(0)
}

func newHPS() *HeaviestSelector {

	hps := New(
		CfgCoordinatorTipselectMinHeaviestBranchUnreferencedBlocksThreshold,
		CfgCoordinatorTipselectMaxHeaviestBranchTipsPerCheckpoint,
		CfgCoordinatorTipselectRandomTipsPerCheckpoint,
		CfgCoordinatorTipselectHeaviestBranchSelectionTimeoutMilliseconds,
	)

	return hps
}

func randBytes(length int) []byte {
	var b []byte
	for i := 0; i < length; i++ {
		b = append(b, byte(rand.Intn(256)))
	}
	return b
}

func randBlockID() hornet.MessageID {
	return randBytes(iotago.BlockIDLength)
}

func newMetadata(parents hornet.MessageIDs) (*inx.BlockMetadata, hornet.MessageID) {
	blockID := randBlockID()
	return &inx.BlockMetadata{
		BlockId: inx.NewBlockId(blockID.ToArray()),
		Parents: utils.INXBlockIDsFromBlockIDs(parents),
		Solid:   true,
	}, blockID
}

func TestHeaviestSelector_SelectTipsChain(t *testing.T) {
	hps := newHPS()

	// create a chain
	lastBlockID := hornet.NullMessageID()
	for i := 1; i <= numTestBlocks; i++ {
		metadata, blockID := newMetadata(hornet.MessageIDs{lastBlockID})
		hps.OnNewSolidBlock(metadata)
		lastBlockID = blockID
	}

	tips, err := hps.SelectTips(1)
	assert.NoError(t, err)
	assert.Len(t, tips, 1)

	// check if the tip on top was picked
	assert.ElementsMatch(t, lastBlockID, tips[0])

	// check if trackedBlocks are resetted after tipselect
	assert.Len(t, hps.trackedBlocks, 0)
}

func TestHeaviestSelector_CheckTipsRemoved(t *testing.T) {
	hps := newHPS()

	count := 8

	blockIDs := make(hornet.MessageIDs, count)
	for i := 0; i < count; i++ {
		metadata, blockID := newMetadata(hornet.MessageIDs{hornet.NullMessageID()})
		hps.OnNewSolidBlock(metadata)
		blockIDs[i] = blockID
	}

	// check if trackedBlocks match the current count
	assert.Len(t, hps.trackedBlocks, count)

	// check if the current tips match the current count
	list := hps.tipsToList()
	assert.Len(t, list.blocks, count)

	// issue a new block that references the old ones
	metadata, blockID := newMetadata(blockIDs)
	hps.OnNewSolidBlock(metadata)

	// old tracked blockIDs should remain, plus the new one
	assert.Len(t, hps.trackedBlocks, count+1)

	// all old tips should be removed, except the new one
	list = hps.tipsToList()
	assert.Len(t, list.blocks, 1)

	// select a tip
	tips, err := hps.SelectTips(1)
	assert.NoError(t, err)
	assert.Len(t, tips, 1)

	// check if the tip on top was picked
	assert.ElementsMatch(t, blockID, tips[0])

	// check if trackedBlocks are resetted after tipselect
	assert.Len(t, hps.trackedBlocks, 0)

	list = hps.tipsToList()
	assert.Len(t, list.blocks, 0)
}

func TestHeaviestSelector_SelectTipsChains(t *testing.T) {
	hps := newHPS()

	numChains := 2
	lastBlockIDs := make(hornet.MessageIDs, 2)
	for i := 0; i < numChains; i++ {
		lastBlockIDs[i] = hornet.NullMessageID()
		for j := 1; j <= numTestBlocks; j++ {
			metadata, blockID := newMetadata(hornet.MessageIDs{lastBlockIDs[i]})
			hps.OnNewSolidBlock(metadata)
			lastBlockIDs[i] = blockID
		}
	}

	// check if all blocks are tracked
	assert.Equal(t, numChains*numTestBlocks, hps.TrackedBlocksCount())

	tips, err := hps.SelectTips(2)
	assert.NoError(t, err)
	assert.Len(t, tips, 2)

	// check if the tips on top of both branches were picked
	assert.ElementsMatch(t, lastBlockIDs, tips)

	// check if trackedBlocks are resetted after tipselect
	assert.Len(t, hps.trackedBlocks, 0)
}

func BenchmarkHeaviestSelector_OnNewSolidBlock(b *testing.B) {
	hps := newHPS()

	blockIDs := hornet.MessageIDs{hornet.NullMessageID()}
	blocks := make([]*inx.BlockMetadata, numBenchmarkBlocks)
	for i := 0; i < numBenchmarkBlocks; i++ {
		tipCount := 1 + rand.Intn(7)
		if tipCount > len(blockIDs) {
			tipCount = len(blockIDs)
		}
		tips := make(hornet.MessageIDs, tipCount)
		for j := 0; j < tipCount; j++ {
			tips[j] = blockIDs[rand.Intn(len(blockIDs))]
		}
		tips = tips.RemoveDupsAndSortByLexicalOrder()

		blocks[i], blockIDs[i] = newMetadata(tips)
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		hps.OnNewSolidBlock(blocks[i%numBenchmarkBlocks])
	}
	hps.Reset()
}
