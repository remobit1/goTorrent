package main

import (
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"math/rand"
	"net/url"
	"time"
)

func main() {
	message := make([]byte, 68)
	MyPeerID := generatePeerID()
	infoHash := generatePeerID()

	copy(message[0:], string(rune(19)))
	copy(message[1:20], "BitTorrent protocol")
	binary.BigEndian.PutUint64(message[20:28], uint64(0))
	copy(message[28:48], infoHash)
	copy(message[48:], MyPeerID)

	fmt.Println(len("BitTorrent protocol"))

	fmt.Print(message)

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
