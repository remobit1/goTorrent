package main

import (
	"fmt"

	"github.com/remobit1/goTorrent/client"
)

func main() {
	name := `files\Oshi no Ko - 05 (720p)(3B609D89)-Erai-raws[TGx].torrent`
	torrent := client.AddTorrent(name)

	tRequest := torrent.Data.CreateTrackerRequest(torrent.Hash)

	torrent.Announce(tRequest)
	fmt.Printf("We grabbed %v peers in total \n", len(torrent.Peers))
	for _, peer := range torrent.Peers {
		fmt.Println(peer)
	}
	torrent.Start()

}
