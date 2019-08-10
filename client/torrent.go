package client

// Will handle torrent file ingestion, marshaling, and eventually handling of the sent pieces assembly in the correct order to form the torrent.

import (
	"crypto/sha1"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"

	"github.com/zeebo/bencode"
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

	_, err = hash.Write(stream)
	if err != nil {
		log.Printf("Unable to hash given stream: %s", err.Error())
		return nil
	}

	return hash.Sum(nil)
}

// CreateTrackerRequest creates an initial tracker request
func (info *MetaInfo) CreateTrackerRequest(hash []byte) *TrackerRequest {
	request := TrackerRequest{
		InfoHash: hash,
		PeerID:   MyPeerID,
		Port:     port,
		Compact:  1,
	}

	return &request

}

// AddPeers takes a slice of peer addresses, creates individual peer objects from them and
// appends them to the Peers field in the torrent struct.
func (torrent *Torrent) AddPeers(addresses []string) {
	for _, address := range addresses {
		newPeer := Peer{
			Address: address,
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
func (torrent *Torrent) Start() {
	for _, peer := range torrent.Peers {
		conn := peer.Handshake(torrent.Hash)
		go peer.HandlePeer(conn, torrent)

	}

	for torrent.torrentNotComplete() {
		torrent.Update()
	}

	torrent.Close()
	fmt.Println("Torrent is done!")
}

// Update checks torrent state then sends appropriate request to peers for missing pieces
func (torrent *Torrent) Update() {
	var wantedPieces []Piece

	for _, piece := range torrent.Pieces {
		if piece.Complete {
			continue
		}
		// create a slice of only unfinished pieces.
		wantedPieces = append(wantedPieces, piece)
	}
	wantedPieces = torrent.arrangePiecesBasedOnRarity(wantedPieces)

	for _, piece := range wantedPieces {
		if len(piece.AvailablePeers) > 0 {
			piece.AvailablePeers[0].reqChannel <- createRequest(&piece)
		}
	}
	return
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

// Close cleans up any outstanding peer connections
func (torrent *Torrent) Close() {
	for _, peer := range torrent.Peers {
		peer.closeConn <- 0
	}
	return
}
