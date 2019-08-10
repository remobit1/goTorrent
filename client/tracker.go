package client

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/zeebo/bencode"
)

const (
	started   string = "started"
	stopped   string = "stopped"
	completed string = "completed"
)

// Will handle tracker requests and updates, i.e. methods that relate to interacting with the tracker

// Announce initiates the first network call to the Tracker for both the UDP and TCP protocol
func (torrent *Torrent) Announce(request *TrackerRequest) {
	/* var announceList []string
	var addressesOfPeers []string

	for _, urls := range torrent.Data.AnnounceList {
		for _, url := range urls {
			url = strings.Replace(url, "udp://", "", 1)
			url = strings.Replace(url, "/announce", "", 1)
			fmt.Println(url)
			announceList = append(announceList, url)
		}
	}

	if torrent.TrackerProtocol == "udp" {
		for _, announceURL := range announceList {
			for _, url := range request.announceUDP(announceURL) {
				addressesOfPeers = append(addressesOfPeers, url)
			}

		}
		torrent.AddPeers(addressesOfPeers)
		return
	}

	for _, announceURL := range announceList {
		for _, url := range request.announceTCP(announceURL) {
			addressesOfPeers = append(addressesOfPeers, url)
		}

	}
	torrent.AddPeers(addressesOfPeers)
	return */

	if torrent.TrackerProtocol == "udp" {
		addressesOfPeers := request.announceUDP(torrent.Data.Announce)
		torrent.AddPeers(addressesOfPeers)
		return
	}
	addressesOfPeers := request.announceTCP(torrent.Data.Announce)
	torrent.AddPeers(addressesOfPeers)
	return
}

/*  announceUDP makes a udp call to the tracker instead of tcp. It sends the connect
message, then parses the response and sends the announce to get the initial list
of peers.
*/
func (request *TrackerRequest) announceUDP(announceURL string) []string {

	raddr, err := net.ResolveUDPAddr("udp", announceURL)
	if err != nil {
		log.Fatalf("Failed to resolve provided udp address: %s \n", err.Error())
	}
	conn, err := net.DialUDP("udp", nil, raddr)
	if err != nil {
		log.Fatalf("Failed to associate underlying socket to read to and from UDP address: %s \n", err.Error())
	}

	defer conn.Close()
	// create a random unique transactionID
	transactionID := func() uint32 {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		return r.Uint32()
	}
	/*  Create a []byte that can hold the 16 bytes that make
	up the initial connect request:
	Offset  Size            Name            Value
	0       32-bit integer  action          0 // connect
	4       32-bit integer  transaction_id
	8       64-bit integer  connection_id
	16
	*/
	req := make([]byte, 16)
	binary.BigEndian.PutUint64(req[0:], 0x41727101980)
	binary.BigEndian.PutUint32(req[8:], 0)
	binary.BigEndian.PutUint32(req[12:], transactionID())

	/* send the request through the udp connection we created
	   earlier to the tracker
	*/
	response, err := sendUDPRequest(conn, req)
	if err != nil {
		fmt.Println(err.Error())
		return nil
	}

	/*  grab the connectionID from the response to send back to the
	tracker in the upcoming announce request.
	*/
	connectionID := response[8:16]

	url := strings.Split(conn.LocalAddr().String(), ":")
	request.Port, err = strconv.Atoi(url[len(url)-1])
	if err != nil {
		fmt.Println(err.Error())
	}

	req = make([]byte, 98)

	/*  Create a new request that can hold the 98 bytes that make
	up our announce:
	Offset  Size   			Name    Value
	0       64-bit integer  connection_id
	8       32-bit integer  action          1 // announce
	12      32-bit integer  transaction_id
	16      20-byte string  info_hash
	36      20-byte string  peer_id
	56      64-bit integer  downloaded
	64      64-bit integer  left
	72      64-bit integer  uploaded
	80      32-bit integer  event           0 // 0: none; 1: completed; 2: started; 3: stopped
	84      32-bit integer  IP address      0 // default
	88      32-bit integer  key
	92      32-bit integer  num_want        -1 // default
	96      16-bit integer  port
	98
	*/
	binary.BigEndian.PutUint64(req[0:], binary.BigEndian.Uint64(connectionID))
	binary.BigEndian.PutUint32(req[8:], 1)
	binary.BigEndian.PutUint32(req[12:], transactionID())
	copy(req[16:36], request.InfoHash)
	copy(req[36:56], request.PeerID)
	binary.BigEndian.PutUint64(req[56:], convertIntToUint64(request.Downloaded))
	binary.BigEndian.PutUint64(req[64:], convertIntToUint64(request.Left))
	binary.BigEndian.PutUint64(req[72:], convertIntToUint64(request.Uploaded))
	binary.BigEndian.PutUint32(req[80:], 2)
	binary.BigEndian.PutUint32(req[84:], 0)
	binary.BigEndian.PutUint32(req[88:], 0)
	binary.BigEndian.PutUint32(req[92:], convertIntToUint32(-1))
	binary.BigEndian.PutUint16(req[96:], convertIntToUint16(request.Port))

	response, err = sendUDPRequest(conn, req)
	if err != nil {
		fmt.Printf("Udp request failed to send: %s \n", err.Error())
	}

	var peers []string
	// Validate that transaction IDs match and the action is 1 (announce), then parse the returned peers
	if string(req[12:16]) == string(response[4:8]) && binary.BigEndian.Uint32(response[:4]) == 1 {
		fmt.Println("Transaction Ids match")
		fmt.Printf("There are %v seeders right now. \n", int32(binary.BigEndian.Uint32(response[16:20])))
		peers = parsePeers(response)
	}

	return peers

}

func sendUDPRequest(conn *net.UDPConn, request []byte) ([]byte, error) {

	rdr := bytes.NewReader(request)

	response := make([]byte, 8192)

	n, err := io.Copy(conn, rdr)

	if err != nil {
		fmt.Printf("Unable to copy request to UDP raddr: %s \n", err.Error())

	}

	fmt.Printf("%v bytes copied to UDP connection \n", n)

	for {
		deadline := time.Now().Add(time.Second * time.Duration(15))
		err := conn.SetReadDeadline(deadline)
		if err != nil {
			fmt.Println(err.Error())
		}
		nRead, _, err := conn.ReadFrom(response)

		if err != nil {
			fmt.Printf("Unable to read from UDP connection: %s \n", err.Error())
			break
		}

		if nRead > 0 {
			return response, nil
		}

	}

	err = errors.New("Unable to establish communication")
	return nil, err
}

/*	WARNING: This method has yet to be tested.
	The first six torrent files I tried were either
	utilizing udp for communication to the tracker
	or even if they were tcp, had 0 peers. I implemented
	the necessary functionality in a hackish manner towards
	the end. I'm moving on for now, but eventually I'll
	need to come back and test this.
*/
func (request *TrackerRequest) announceTCP(announceURL string) []string {
	request.Event = started

	objURL, err := url.Parse("http://" + announceURL + "/announce")

	if err != nil {
		fmt.Printf("Cannot parse provide url: %s \n", err.Error())
	}

	params := url.Values{}

	params.Add("info_hash", string(request.InfoHash))
	params.Add("peer_id", string(request.PeerID))
	params.Add("port", strconv.Itoa(request.Port))
	params.Add("uploaded", strconv.Itoa(request.Uploaded))
	params.Add("downloaded", strconv.Itoa(request.Downloaded))
	params.Add("left", strconv.Itoa(request.Left))
	params.Add("compact", strconv.Itoa(request.Compact))
	params.Add("event", string(request.Event))

	objURL.RawQuery = params.Encode()

	response, err := http.Get(objURL.String())

	if err != nil {
		fmt.Printf("Announce to Tracker failed: %s \n", err.Error())
	}

	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		fmt.Printf("Coudn't read response body: %s \n", err.Error())
	}
	var trackerResponse TrackerResponse
	bencode.DecodeBytes(body, &trackerResponse)

	//Doing something hackish temporarily
	resp := make([]byte, (len([]byte(trackerResponse.Peers)) + 20))
	copy(resp[20:], []byte(trackerResponse.Peers))
	// if the length of newly created response indicates at least one peer (over 30 bytes)
	// send response to have its peers parsed
	if len(resp) > 30 {
		peers := parsePeers(resp)
		for _, peer := range peers {
			fmt.Println(peer)
		}
		return peers
	}

	fmt.Println("No peers included in tracker response")
	return nil
}

func isUDP(u *url.URL) bool {
	if u.Scheme == "udp" {
		return true
	}
	return false
}

func parsePeers(response []byte) []string {
	var peers []string

	/* we're going to create two quick helper functions to calculate the
	   index ranges for the peers in the byte array. From the spec, we know
	   that every peer IP is 20 + (6*n) while every associated port is 4 bytes
	   after it, or 24 + (6*n)
	*/
	ipIndex := func(x int) int {
		return 20 + (6 * x)
	}

	portIndex := func(x int) int {
		return 24 + (6 * x)
	}

	// Loop until we get to an empty IP address. Grab IPs and strings then combine them.
	for i := 0; binary.BigEndian.Uint32(response[ipIndex(i):portIndex(i)]) != uint32(0); i++ {

		peerIP := make(net.IP, 4)
		binary.BigEndian.PutUint32(peerIP, binary.BigEndian.Uint32(response[ipIndex(i):portIndex(i)]))

		peerPort := strconv.FormatUint(uint64(binary.BigEndian.Uint16(response[portIndex(i):(portIndex(i)+2)])), 10)

		if peerIP.String() == "" {
			break
		}

		peer := peerIP.String() + ":" + peerPort
		peers = append(peers, peer)

	}

	return peers
}
