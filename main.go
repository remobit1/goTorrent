package main

import (
	"JTorrent/client"
)

func main() {
	name := `files\The Lego Movie 2 The Second Part (2019) [BluRay] [1080p] [YTS.LT].torrent`
	torrent := client.AddTorrent(name)

	tRequest := torrent.Data.CreateTrackerRequest(client.MyPeerID)

	torrent.Announce(tRequest)

	torrent.Start()

}
