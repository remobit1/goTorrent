package main

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/zeebo/bencode"
)

// MetaInfo represents the information .torrent file that stores the information needed to download a torrent.
type MetaInfo struct {
	Announce     string         `bencode:"announce"`
	AnnounceList [][]string     `bencode:"announce-list,omitempty"`
	CreationDate int            `bencode:"creation date,omitempty"`
	Comment      string         `bencode:"comment,omitempty"`
	CreatedBy    string         `bencode:"created by,omitempty"`
	Encoding     string         `bencode:"encoding,omitempty"`
	Info         InfoDictionary `bencode:"info"`
}

// The InfoDictionary is a dictionary that describes the file(s) of the torrent.
type InfoDictionary struct {
	PieceLength int                `bencode:"piece length"`
	Pieces      bencode.RawMessage `bencode:"pieces"`
	Private     int                `bencode:"private,omitempty"`
	Name        string             `bencode:"name"`
	Files       []File             `bencode:"files,omitempty"`
	//Length      int    `bencode:"length,omitempty"`
	//Md5sum      string `bencode:"md5sum,omitempty"`
}

// A File represents the info dictionary of a torrent in single-file mode, as well as the multipe files in multi-file mode
type File struct {
	Length int      `bencode:"length"`
	Md5sum string   `bencode:"md5sum,omitempty"`
	Path   []string `bencode:"path"` // Bencoded list of strings that represent the path and filename
}

func main() {
	f, err := os.Open("Marvel's Avengers (v1.3.3-141640, MULTi15).torrent")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	testy := MetaInfo{}
	stream, err := ioutil.ReadAll(f)
	if err != nil {
		log.Fatal(err)
	}
	l, err := os.Open("Marvel's Avengers (v1.3.3-141640, MULTi15).torrent")
	if err != nil {
		log.Fatal(err)
	}
	defer l.Close()
	fmt.Println(string(stream))
	bencoder := bencode.NewDecoder(l)
	err = bencoder.Decode(&testy)

	if err != nil {
		log.Fatal(err)
	}

	/*fmt.Println(string(stream[(len(stream) - 191):(len(stream) - 1)]))
	fmt.Println(stream[(len(stream) - 21):(len(stream) - 1)])
	h := sha1.New()
	h.Write(stream[(len(stream) - 191):(len(stream) - 1)])
	byteHash := h.Sum(nil)
	dst := make([]byte, hex.EncodedLen(len(byteHash)))
	hex.Encode(dst, byteHash)
	fmt.Println(string(dst)) */

	encodedAgain, err := bencode.EncodeBytes(testy.Info)

	if err != nil {
		log.Fatal(err)
	}

	n := sha1.New()
	n.Write(encodedAgain)
	bHash := n.Sum(nil)
	dest := make([]byte, hex.EncodedLen(len(bHash)))
	hex.Encode(dest, bHash)
	fmt.Println(string(dest))

}
