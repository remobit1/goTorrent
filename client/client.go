package client

/* handles routing and organization
1. Init should do the following: check a designated folder for torrent files
2. create necessary tcp and udp connections based on the 'announce' in the various info dictionary
3. start seeding and leeching for each individual torrent
4. Wait for more torrents to be added.
*/
import (
	"crypto/sha1"
	"encoding/binary"
	"math/rand"
	"net"
	"time"
)

var (
	port     int
	listener net.Listener
	// MyPeerID is generated at startup to be used to identify the client to trackers and peers
	MyPeerID []byte
)

func init() {
	MyPeerID = generatePeerID()
}

func generatePeerID() []byte {
	hash := sha1.New()
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	hashed := make([]byte, 4)
	binary.LittleEndian.PutUint32(hashed, r.Uint32())

	hash.Write(hashed)

	return hash.Sum(nil)
}
