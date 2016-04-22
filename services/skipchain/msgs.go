package skipchain

import (
	"github.com/dedis/cothority/lib/crypto"
	"github.com/dedis/cothority/lib/network"
	"github.com/satori/go.uuid"
)

func init() {
	var msgs = []interface{}{
		&ProposeSkipBlock{},
		&ProposedSkipBlockReply{},
		&GetUpdateChain{},
		&GetUpdateChainReply{},
		&PropagateSkipBlock{},
		&ForwardSignature{},
		&SkipBlockData{},
	}
	for _, m := range msgs {
		network.RegisterMessageType(m)
	}
}

type VerifierId uuid.UUID
type RosterId uuid.UUID

// This file holds all messages that can be sent to the SkipChain,
// both from the outside and between instances of this service

// External calls

// RequestNewBlock - Requests a new skipblock to be appended to
// the given SkipBlock. If the given SkipBlock has Index 0 (which
// is invalid), a new SkipChain will be created.
// The AppId will be used to call the corresponding verification-
// routines who will have to sign off on the new Tree.
type ProposeSkipBlock struct {
	Latest   crypto.HashID
	Proposed SkipBlock
}

// ProoposedReply - returns the signed SkipBlock with updated backlinks
type ProposedSkipBlockReply struct {
	Previous SkipBlock
	Latest   SkipBlock
}

// GetUpdateChain - the client sends the hash of the last known
// Skipblock and will get back a list of all necessary SkipBlocks
// to get to the latest.
type GetUpdateChain struct {
	Latest crypto.HashID
}

// GetUpdateChainRet - returns the shortest chain to the current SkipBlock,
// starting from the SkipBlock the client sent
type GetUpdateChainReply struct {
	Update []SkipBlock
}

// Internal calls

// PropagateSkipBlock sends a newly signed SkipBlock to all members of
// the Cothority
type PropagateSkipBlock struct {
	SkipBlock SkipBlock
}

// ForwardSignature asks this responsible for a SkipChain to sign off
// a new ForwardLink. This will probably be sent to all members of any
// SkipChain-definition at time 'n'
type ForwardSignature struct {
	ToUpdate crypto.HashID
	Latest   SkipBlock
}
