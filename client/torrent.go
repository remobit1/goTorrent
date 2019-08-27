package client

// Will handle torrent file ingestion, marshaling, and eventually handling of the sent pieces assembly in the correct order to form the torrent.

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/url"
	"os"
	"time"

	"github.com/zeebo/bencode"
)

const (
	maxPeers = 5
)

// AddTorrent returns a MetaInfo struct
func AddTorrent(path string) *Torrent {
	file, err := os.Open(path)
	if err != nil {
		log.Printf("Error opening file: %s", err.Error())
	}

	stream, err := ioutil.ReadAll(file)

	if err != nil {
		log.Printf("Error reading file: %s", err.Error())
	}

	info := MetaInfo{}

	bencode.DecodeBytes(stream, &info)

	protocol := "tcp"

	u, err := url.Parse(info.Announce)
	if err != nil {
		fmt.Printf("Unable to parse url: %s", err.Error())
	}

	info.Announce = u.Host

	if isUDP(u) {
		protocol = "udp"
	}
	t := Torrent{Path: path, Data: info, TrackerProtocol: protocol}
	t.Hash = info.HashInfo()

	t.splitPieces()

	if _, err := os.Stat("downloadedTorrent"); err == nil {
		t.Path = "downloadedTorrent"
	} else {
		_, err := os.Create("downloadedTorrent")
		if err != nil {
			fmt.Println(err.Error())
		}
	}

	return &t
}

// HashInfo hashes the info section of the MetaInfo file in preparation for a tracker request
func (info *MetaInfo) HashInfo() []byte {
	hash := sha1.New()
	stream, err := bencode.EncodeBytes(info.Info)
	if err != nil {
		log.Printf("Unable to encode object: %s", err.Error())
		return nil
	}
	streamer := bytes.NewReader(stream)
	_, err = io.Copy(hash, streamer)
	if err != nil {
		log.Printf("Unable to hash given stream: %s", err.Error())
		return nil
	}

	infoHash := hash.Sum(nil)

	return infoHash
}

// CreateTrackerRequest creates an initial tracker request
func (info *MetaInfo) CreateTrackerRequest(hash []byte) *TrackerRequest {
	// create a random unique transactionID
	transactionID := func() uint32 {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		return r.Uint32()
	}

	ID := make([]byte, 4)

	binary.BigEndian.PutUint32(ID[:], transactionID())
	request := TrackerRequest{
		InfoHash:      hash,
		PeerID:        MyPeerID,
		Port:          port,
		Compact:       1,
		TransactionID: ID,
	}

	return &request

}

// AddPeers takes a slice of peer addresses, creates individual peer objects from them and
// appends them to the Peers field in the torrent struct.
func (torrent *Torrent) AddPeers(addresses []string) {
	for _, addr := range addresses {
		newPeer := Peer{
			address: addr,
		}
		torrent.Peers = append(torrent.Peers, newPeer)
	}
}

// helper function to split string of torrent piece hashes into slice of said hashes.
func (torrent *Torrent) splitPieces() {
	notSplit := torrent.Data.Info.Pieces
	var indexes []int
	var splitPieces []string
	/* append to slice in intervals of (20*i) to (20*(i+1))
	   to separate every 20 byte hash */
	for i := 0; len(notSplit[:(20*(i+1))]) < len(notSplit); i++ {
		splitPieces = append(splitPieces, string(notSplit[(20*i):(20*(i+1))]))
		indexes = append(indexes, i)
	}

	/* create a corresponding piece struct for every hash
	   and append to torrent.Pieces */
	for i, hash := range splitPieces {
		piece := Piece{
			Hash:  []byte(hash),
			Index: indexes[i],
		}
		piece.prepBlocks(torrent.Data.Info.PieceLength)
		torrent.Pieces = append(torrent.Pieces, piece)
	}
}

func (piece *Piece) prepBlocks(pieceLength int) {
	amountOfBlocks := pieceLength / 16384

	for i := 0; amountOfBlocks > i; i++ {
		blk := Block{
			Offset: i * 16384,
		}

		piece.Blocks = append(piece.Blocks, blk)
	}
}

// Start starts the torrent download
func (torrent *Torrent) Start(laddr *net.TCPAddr) {
	clientState.torrents = append(clientState.torrents, *torrent)

	for _, peer := range torrent.Peers {
		fmt.Println("connecting to peer...")
		conn, err := peer.initiateConnection(torrent.Hash)
		if err != nil {
			fmt.Printf("Unable to establish connection with peer: %s \n", err.Error())
		}
		go peer.handlePeerConnection(conn)
	}

	for torrent.torrentNotComplete() {
		torrent.Update()
	}

	torrent.StopDownloading()
	fmt.Println("Torrent is done!")
}

// Update checks torrent state then sends appropriate request to peers for missing pieces
func (torrent *Torrent) Update() {

}

//StopDownloading closes all current connections with peers after a torrent is finished.
func (torrent *Torrent) StopDownloading() {
	for _, peer := range torrent.Peers {
		connection := *peer.conn
		err := connection.Close()
		if err != nil {
			fmt.Printf("Unable to close peer connection: %s \n", err.Error())
		}
	}
}

func (torrent *Torrent) arrangePiecesBasedOnRarity(pieces []Piece) []Piece {
	var orderedPieces []Piece
	for _, piece := range pieces {
		for _, peer := range torrent.Peers {
			if peer.Bitfield != nil && peer.Bitfield[piece.Index] == 1 {
				piece.AvailablePeers = append(piece.AvailablePeers, &peer)
			}
		}
		orderedPieces = append(orderedPieces, piece)
	}

	return orderedPieces
}

func (torrent *Torrent) torrentNotComplete() bool {
	for _, piece := range torrent.Pieces {
		if !piece.Complete {
			return true
		}
	}

	return false
}

func (torrent *Torrent) writeToFile(data []byte, offset int) {
	file, err := os.Open(torrent.Path)
	if err != nil {
		fmt.Println(err.Error())
	}

	_, err = file.Seek(int64(offset), 0)

	if err != nil {
		fmt.Println(err.Error())
	}

	_, err = file.Write(data)

	if err != nil {
		fmt.Println(err.Error())
	}

	return
}
