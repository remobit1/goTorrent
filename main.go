package main

import (
	"goTorrent/client"
)

func main() {
	name := `files\The Lego Movie 2 The Second Part (2019) [BluRay] [1080p] [YTS.LT].torrent`
	torrent := client.AddTorrent(name)

	tRequest := torrent.Data.CreateTrackerRequest(torrent.Hash)

	torrent.Announce(tRequest)

	torrent.Start()

}
