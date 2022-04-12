package coordinator

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gohornet/hornet/pkg/model/hornet"
	"github.com/gohornet/hornet/pkg/model/milestone"
	"github.com/gohornet/hornet/plugins/debug"
	iotago "github.com/iotaledger/iota.go/v3"
	"github.com/iotaledger/iota.go/v3/nodeclient"
)

const (
	// NodeAPIRouteDebugComputeWhiteFlag is the debug route to compute the white flag confirmation for the cone of the given parents.
	// POST computes the white flag confirmation.
	NodeAPIRouteDebugComputeWhiteFlag = "/api/plugins/debug/v1/whiteflag"
)

// NewDebugNodeAPIClient returns a new DebugNodeAPIClient with the given BaseURL.
func NewDebugNodeAPIClient(baseURL string, opts ...nodeclient.ClientOption) *DebugNodeAPIClient {
	return &DebugNodeAPIClient{Client: nodeclient.New(baseURL, opts...)}
}

// DebugNodeAPIClient is a client for node HTTP REST APIs.
type DebugNodeAPIClient struct {
	*nodeclient.Client
}

// WhiteFlag is the debug route to compute the white flag confirmation for the cone of the given parents.
// This function returns the merkle tree hash calculated by the node.
func (api *DebugNodeAPIClient) WhiteFlag(index milestone.Index, timestamp uint32, parents hornet.MessageIDs, lastMilestoneID iotago.MilestoneID) (*MilestoneMerkleProof, error) {

	req := &debug.ComputeWhiteFlagMutationsRequest{
		Index:           index,
		Timestamp:       timestamp,
		Parents:         parents.ToHex(),
		LastMilestoneID: iotago.EncodeHex(lastMilestoneID[:]),
	}

	res := &debug.ComputeWhiteFlagMutationsResponse{}

	if _, err := api.Do(context.Background(), http.MethodPost, NodeAPIRouteDebugComputeWhiteFlag, req, res); err != nil {
		return nil, err
	}

	inclusionMerkleProofBytes, err := iotago.DecodeHex(res.InclusionMerkleProof)
	if err != nil {
		return nil, err
	}

	if len(inclusionMerkleProofBytes) != iotago.MilestoneMerkleProofLength {
		return nil, fmt.Errorf("unknown merkle tree hash length (%d)", len(inclusionMerkleProofBytes))
	}

	pastConeMerkleProofBytes, err := iotago.DecodeHex(res.InclusionMerkleProof)
	if err != nil {
		return nil, err
	}

	if len(pastConeMerkleProofBytes) != iotago.MilestoneMerkleProofLength {
		return nil, fmt.Errorf("unknown merkle tree hash length (%d)", len(pastConeMerkleProofBytes))
	}

	merkleProof := &MilestoneMerkleProof{
		PastConeMerkleProof:  &MerkleTreeHash{},
		InclusionMerkleProof: &MerkleTreeHash{},
	}
	copy(merkleProof.InclusionMerkleProof[:], inclusionMerkleProofBytes)
	copy(merkleProof.PastConeMerkleProof[:], pastConeMerkleProofBytes)
	return merkleProof, nil
}
