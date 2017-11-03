package darc

import (
	"gopkg.in/dedis/crypto.v0/abstract"
)

// ID is the identity of a Darc - which is the sha256 of its protobuf representation
// over invariant fields [Owners, Users, Version, Description]. Signature is excluded.
// An evolving Darc will change its identity.
type ID []byte

// Role indicates if this is an Owner or a User Policy.
type Role int

const (
	// Owner has the right to evolve the Darc to a new version.
	Owner Role = iota
	// User has the right to sign on behalf of the Darc for
	// whatever decision is asked for.
	User
)

// PROTOSTART
//
// option java_package = "ch.epfl.dedis.proto";
// option java_outer_classname = "DarcProto";

// ***
// These are the messages used in the API-calls
// ***

// Darc is the basic structure representing an access control. A Darc can evolve in the way that
// a new Darc points to the previous one and is signed by the owner(s) of the previous Darc.
type Darc struct {
	// Identities who are allowed to evolve this Darc.
	Owners *[]*Identity
	// Identities who can perform actions (write/read) with data on a skipchain.
	Users *[]*Identity
	// Version should be monotonically increasing over the evolution of a Darc.
	Version uint32
	// Description is a free-form field that can hold any data as required by the user.
	// Darc itself will never depend on any of the data in here.
	Description *[]byte
	// Signature is calculated over the protobuf representation of [Owner, Users, Version, Description]
	// and needs to be created by an Owner from the previous valid Darc.
	Signature *Signature
}

// Identity is a generic structure can be either an Ed25519 public key or a Darc
type Identity struct {
	// Darc identity
	Darc *IdentityDarc
	// Public-key identity
	Ed25519 *IdentityEd25519
}

// IdentityEd25519 holds a Ed25519 public key (Point)
type IdentityEd25519 struct {
	Point abstract.Point
}

// IdentityDarc is a structure that points to a Darc with a given DarcID on a skipchain
type IdentityDarc struct {
	ID ID
}

// Signature is a signature on a Darc to accept a given decision.
// can be verified using the appropriate identity.
type Signature struct {
	// The signature itself
	Signature []byte
	// Represents the path to get up to information to be able to verify this signature
	SignaturePath SignaturePath
}

// SignaturePath is a struct that holds information necessary for signature verification
type SignaturePath struct {
	// Darc(s) that justify the right of the signer to push a new Darc
	Darcs *[]*Darc
	// the Idenity (public key or another Darc) of the signer
	Signer Identity
	// Is the signer Owner of a Darc or an user
	Role Role
}

// Signer is a generic structure that can hold different types of signers
// TODO Make it an interface
type Signer struct {
	Ed25519 *Ed25519Signer
}

// Ed25519Signer holds a public and private keys necessary to sign Darcs
type Ed25519Signer struct {
	Point  abstract.Point
	Secret abstract.Scalar
}
