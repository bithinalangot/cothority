// Catena implements the simplest possible example of how to set up the
// logread-service. It shall be used as a source for copy/pasting.
package main

import (
	"bytes"

	"strings"

	"github.com/dedis/cothority/skipchain"
	"github.com/dedis/logread"
	"gopkg.in/dedis/crypto.v0/cipher/sha3"
	"gopkg.in/dedis/crypto.v0/random"
	"gopkg.in/dedis/onet.v1/app"
	"gopkg.in/dedis/onet.v1/log"
)

// publicTomlData is defined as a constant here - depending on the setup
// chosen to link the docker-containers, the addresses might need to be
// adjusted.
// Even though the ports given here are 7002, 7004 and 7006, the communication
// between the app and the conodes happens on ports 7003, 7005 and 7007.
const publicTomlData = `[[servers]]
  Address = "tcp://127.0.0.1:7002"
  Public = "mkA0EYEqjNMC+jVxtVCPaUI3oWjwt5TNDK8bpXgHL0Q="
  Description = "Conode_1"
[[servers]]
  Address = "tcp://127.0.0.1:7004"
  Public = "ljojlp5FKO05HZRTy5aVAV5kaWWN+0vxfIHLZ3dW79Q="
  Description = "Conode_2"
[[servers]]
  Address = "tcp://127.0.0.1:7006"
  Public = "imd1y9Mp3vey1GdyFy6gk+w0XAuAN7wO34u5rcKiIHI="
  Description = "Conode_3"
`

// dummy invoice data that will be stored on the skipchain.
var invoiceData = []byte("company x orders y at z for _ CHF")

// main creates a new chain, stores the invoiceData on it, and then retrieves
// the data.
func main() {
	// Don't show program-lines - set to 1 or higher for mroe debugging
	// messages
	log.SetDebugVisible(0)
	lr, err := setupChains()
	log.ErrFatal(err)

	// Marshalling the logread-structure, so that it can be stored
	// in a file or as a value in a keyvalue storage.
	forStorage, err := lr.Marshal()
	log.ErrFatal(err)
	log.Info("Having", len(forStorage), "bytes for logread-storage")

	// Unmarshalling the logread-structure to access it
	lrLoaded, err := logread.NewLogreadUnmarshal(forStorage)
	fileID, err := writeFile(lrLoaded, invoiceData)
	log.ErrFatal(err)

	// Getting the file off the skipchain
	data, err := readFile(lrLoaded, fileID, "chaincode")
	log.ErrFatal(err)
	if bytes.Compare(invoiceData, data) != 0 {
		log.Fatal("Original data and retrieved data are not the same")
	}
	log.Info("Retrieved data:", string(data))

	// Reading at most 4 read-requests from the start
	requests, err := lrLoaded.GetReadRequests(nil, 4)
	log.ErrFatal(err)
	for _, req := range requests {
		log.Infof("User %s read document %x", req.Reader, req.FileID)
	}
}

// setupChains creates the skipchains needed and adds two users:
//  - admin - with the right to add/remove users
//  - client - with the right to write to the skipchain
//  - chaincode - with the right to read from the skipchain
func setupChains() (lr *logread.Logread, err error) {
	group, err := app.ReadGroupDescToml(strings.NewReader(publicTomlData))
	if err != nil {
		return
	}

	// In the next step we create a new logread-skipchain with an admin-user
	// called 'admin'.
	log.Info("Setting up skipchains")
	lr, err = logread.NewLogread(group.Roster, "admin")
	if err != nil {
		return
	}

	// Now we add two users:
	// client - with write access
	// chaincode - with read access
	log.Info("Adding users")
	if err = lr.AddUser("client", logread.UserWriter); err != nil {
		return
	}
	if err = lr.AddUser("chaincode", logread.UserReader); err != nil {
		return
	}
	return
}

// writeFile stores the data on the skipchain and returns a fileID that can
// be used to retrieve that data. fileID is a unique identifier over all
// the skipchain.
func writeFile(lr *logread.Logread, data []byte) (fileID skipchain.SkipBlockID, err error) {
	// The client stores a file on the skipchain.
	log.Info("Encrypting file and sending it to the skipchain")
	// 1. Create a random symmetric key
	key := random.Bytes(32, random.Stream)
	// 2. Encrypt our data using that key
	cipher := sha3.NewShakeCipher128(key)
	encData := cipher.Seal(nil, data)
	// 3. Encrypt the key using the secret share public key
	encKey, cerr := lr.EncryptKey(key)
	if cerr != nil {
		err = cerr
		return
	}
	// 4. Write the encrypted data with the encrypted key to the skipchain.
	fileID, err = lr.AddFile(encData, encKey, "client")
	return
}

// readFile requests the data from the skipchain under the reader's name. If
// reader is not registered to the skipchain, the function will return an error.
func readFile(lr *logread.Logread, idFile skipchain.SkipBlockID, reader string) ([]byte, error) {
	// Now the chaincode requests access to the file.
	log.Info("Send file-request")
	readRequest, err := lr.RequestFile(idFile, reader)
	if err != nil {
		return nil, err
	}

	// Finally fetch the file (supposing we don't have it yet)
	log.Info("Get re-encrypted key")
	encData, key, err := lr.ReadFile(readRequest)
	if err != nil {
		return nil, err
	}

	// And decrypt it
	log.Info("Decrypt the data")
	cipher := sha3.NewShakeCipher128(key)
	data, err := cipher.Open(nil, encData)
	if err != nil {
		return nil, err
	}
	return data, nil
}
