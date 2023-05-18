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
	"net/url"
	"strconv"
	"time"
)

var (
	// MyPeerID is generated at startup to be used to identify the client to trackers and peers
	MyPeerID    []byte
	clientState Client
)

const (
	port = 4242
)

func init() {
	MyPeerID = generatePeerID()
	portString := strconv.Itoa(port)

	laddr := []byte(":")

	copy(laddr[1:], []byte(portString))

	listeningPort, err := net.ResolveTCPAddr("tcp", string(laddr))
	if err != nil {
		panic(err)
	}

	go Listen(listeningPort)

}

func generatePeerID() []byte {
	hash := sha1.New()
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	hashed := make([]byte, 4)
	binary.LittleEndian.PutUint32(hashed, r.Uint32())

	hash.Write(hashed)

	peerID := []byte(url.QueryEscape(string(hash.Sum(nil))))

	return peerID
}
