package logread

import (
	"errors"

	"github.com/dedis/cothority/skipchain"
	"gopkg.in/dedis/onet.v1"
	"gopkg.in/dedis/onet.v1/crypto"
	"gopkg.in/dedis/onet.v1/log"
	"gopkg.in/dedis/onet.v1/network"
)

func init() {
	network.RegisterMessage(&Logread{})
}

// This file holds wrappers around all the basic methods used to set up the
// logread-service.

// Logread holds everything needed to write and read to the skipchain. It can
// be marshaled and unmarshalled to be passed between different methods. The
// marshaled size is about 1kB, independent of the number of files stored.
type Logread struct {
	LatestACL *skipchain.SkipBlock
	LatestWlr *skipchain.SkipBlock
	ACL       *DataACL
	Admin     *Credential
	cl        *Client
}

// NewLogread takes a roster of conodes and a name for the administrator. It
// returns a Logread-structure, or an error if it couldn't set up the
// skipchains.
func NewLogread(r *onet.Roster, name string) (*Logread, error) {
	cl := NewClient()
	skipblockACL, skipblockWLR, admin, cerr := cl.CreateSkipchains(r, name)
	if cerr != nil {
		return nil, cerr
	}
	return &Logread{
		LatestACL: skipblockACL,
		LatestWlr: skipblockWLR,
		ACL:       NewDataACL(skipblockACL.Data),
		Admin:     admin,
		cl:        cl,
	}, nil
}

// NewLogreadUnmarshal takes a slice of bytes and returns a Logread-structure
// or an error if the data couldn't be unmarshalled.
func NewLogreadUnmarshal(data []byte) (*Logread, error) {
	_, lri, err := network.Unmarshal(data)
	if err != nil {
		return nil, err
	}
	lr, ok := lri.(*Logread)
	if !ok {
		return nil, errors.New("this is not a Logread-slice")
	}
	lr.cl = NewClient()
	return lr, nil
}

// Marshal returns a slice of byte of the Logread-structure. The slice of bytes
// can be unmarshalled using NewLogreadUnmarshal
func (lr *Logread) Marshal() ([]byte, error) {
	// Lighten the load of the skipblocks - the data is not needed.
	lr.LatestACL.Data = []byte{}
	lr.LatestWlr.Data = []byte{}
	return network.Marshal(lr)
}

const (
	// UserAdmin has the right to add/remove other users
	UserAdmin = iota
	// UserWriter has the right to add documents, but not to read
	UserWriter
	// UserReader has the right to request access to documents
	UserReader
)

// AddUser adds a new pseudonym with 'name' and 'userType' to the ACL-skipchain.
func (lr *Logread) AddUser(name string, userType int) error {
	switch userType {
	case UserAdmin:
		lr.ACL.Admins.AddPseudo(name)
	case UserWriter:
		lr.ACL.Writers.AddPseudo(name)
	case UserReader:
		lr.ACL.Readers.AddPseudo(name)
	default:
		return errors.New("don't know this type of user")
	}
	reply, err := lr.cl.EvolveACL(lr.LatestACL, lr.ACL, lr.Admin)
	if err != nil {
		return err
	}
	lr.LatestACL = reply.SB
	return nil
}

// DelUser removes all users with name from admins, writers and readers.
// If there is only 1 admin, it will not try to remove that one, as this
// would leave no admin for the acl.
func (lr *Logread) DelUser(name string) error {
	if len(lr.ACL.Admins.List) > 1 {
		// Make sure we don't delete the last admin
		lr.ACL.Admins.DelPseudo(name)
	}
	lr.ACL.Writers.DelPseudo(name)
	lr.ACL.Readers.DelPseudo(name)

	reply, err := lr.cl.EvolveACL(lr.LatestACL, lr.ACL, lr.Admin)
	if err != nil {
		return err
	}
	lr.LatestACL = reply.SB
	return nil
}

// EncryptKey asks the skipchain for the public key of the secret shared key
// and returns the key encrypted with that.
// The key is not sent to the conode.
func (lr *Logread) EncryptKey(key []byte) ([]byte, error) {
	return lr.cl.EncryptKeyRequest(lr.LatestWlr.Roster, key)
}

// AddFile requests a file to be stored on the skipchain. The user 'name' has
// to have write-access to the skipchain, else he won't be able to store
// anything on the skipchain. If the write-operation succeeds, the returned
// SkipBlockID can be used to make a read-request.
//  - encData is encrypted by key
//  - encKey is the Logread.EncryptKey(key)
//  - name is the name of the writer
func (lr *Logread) AddFile(encData, encKey []byte, name string) (skipchain.SkipBlockID, error) {
	writer, _ := lr.ACL.Writers.FindPseudo(name)
	if writer == nil {
		return nil, errors.New("didn't find writer")
	}
	sb, cerr := lr.cl.WriteRequest(lr.LatestWlr, encData, encKey, writer)
	if cerr != nil {
		return nil, cerr
	}
	lr.LatestWlr = sb
	return sb.Hash, nil
}

// FileRequest holds all needed information to retrieve a file once a request
// has been successful.
type FileRequest struct {
	File   skipchain.SkipBlockID
	Read   skipchain.SkipBlockID
	Cred   *Credential
	EncKey []byte
}

// RequestFile asks the skipchain to re-encrypt the symmetric key for the file 'id'
// under the reader's public key. 'name' is the name of the reader and needs to
// have read-access, else the request is denied and an error is returned.
func (lr *Logread) RequestFile(id skipchain.SkipBlockID, name string) (*FileRequest, error) {
	reader, _ := lr.ACL.Readers.FindPseudo(name)
	if reader == nil {
		return nil, errors.New("didn't find reader")
	}
	sb, cerr := lr.cl.ReadRequest(lr.LatestWlr, reader, id)
	if cerr != nil {
		log.Error(cerr)
		return nil, cerr
	}
	lr.LatestWlr = sb
	_, dwI, err := network.Unmarshal(sb.Data)
	if err != nil {
		return nil, err
	}
	dw, ok := dwI.(*DataWlr)
	if !ok {
		return nil, errors.New("didn't get correct skipblock")
	}
	read := dw.Read
	if read.Pseudonym != name {
		return nil, errors.New("got wrong pseudo")
	}
	if crypto.VerifySchnorr(network.Suite, reader.Public, read.File, *read.Signature) != nil {
		return nil, errors.New("Wrong signature")
	}
	return &FileRequest{
		File:   id,
		Read:   sb.Hash,
		EncKey: read.EncKey,
		Cred:   reader,
	}, nil
}

// ReadFile returns the file and the decrypted key from a read-request. It needs
// to contact a conode to get the re-encryption done.
func (lr *Logread) ReadFile(read *FileRequest) (file, key []byte, err error) {
	var cerr onet.ClientError
	file, cerr = lr.cl.GetFile(lr.LatestWlr.Roster, read.File)
	if cerr != nil {
		err = cerr
		return
	}
	key, cerr = lr.cl.DecryptKeyRequest(lr.LatestWlr.Roster, read.Read,
		read.Cred)
	return
}

// GetReadRequests searches the skipchain for requests and returns all found
// requests. if 'start' is nil, the first read-requests are returned. A maximum
// of 'count' requests are returned.
func (lr *Logread) GetReadRequests(start skipchain.SkipBlockID, count int) ([]*ReadDoc, error) {
	if start.IsNull() {
		start = lr.LatestWlr.SkipChainID()
	}
	return lr.cl.GetReadRequests(lr.LatestWlr.Roster, start, count)
}
