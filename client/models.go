package client

/* TODO: GO BACK AND UNEXPORT EVERY STRUCT FIELD THAT ISN'T BEING USED OUTSIDE THIS CLIENT PACKAGE

1. Read into memory a MetaInfo file and UNMarshal it (Check)
2. Generate an InfoHash based on the information and a unique PeerID
3. If done correctly, the tracker will respond with information including a list of Peers.
4. (Future) Scrape the tracker to get updates on peers
5. Initiate a Handshake with peers over TCP with previously generated infoHash and PeerID
6. After the Handshake, communicate with peers via length-prefixed messages.
7. Messages take the form of <length prefix><message ID><payload>. The payload is message dependent.
8. Use request messages to request blocks. Use piece message to send blocks.
9. Use the index, begin and block to assemble blocks once downloaded.
10. Hash the pieces that the blocks form, tell your peers and the trackers about the hashed pieces, and do that until you're done.
11. You have a complete torrent! (I think)

*/

// Torrent contains all necessary information to start downloading a torrent
type Torrent struct {
	Path            string
	Data            MetaInfo
	Hash            []byte
	TrackerProtocol string //whether its tracker uses UDP or TCP
	Peers           []Peer
	Pieces          []Piece
	ConnectedPeers  int
}

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
	Name        string `bencode:"name"`
	PieceLength int    `bencode:"piece length"`
	Pieces      string `bencode:"pieces"`
	Private     int    `bencode:"private,omitempty"`
	Files       []File `bencode:"files,omitempty"`
	Length      int    `bencode:"length,omitempty"`
	Md5sum      string `bencode:"md5sum,omitempty"`
}

// A File represents the info dictionary of a torrent in single-file mode, as well as the multipe files in multi-file mode
type File struct {
	Path   []string `bencode:"path"` // Bencoded list of strings that represent the path and filename
	Length int      `bencode:"length"`
	Md5sum string   `bencode:"md5sum,omitempty"`
}

// A TrackerRequest is a client to tracker GET request
type TrackerRequest struct {
	InfoHash      []byte // URLencoded 20-byte SHA1 hash of the value of the info key from the MetaInfo file, bencoded.
	PeerID        []byte // URLencoded 20-byte string used as a unique ID for the client, and generated by the client at startup.
	Port          int    // Port number the client is listening on
	Uploaded      int    // Total amount of bytes uploaded in base ten ASCII
	Downloaded    int    // Total amount of bytes downloaded  in base ten ASCII
	Left          int    // The number of bytes that still need to be downloaded to reach 100%
	Compact       int    // A setting that tells whether the client accepts a compact response or not using 0 and 1
	Event         string // Must be one of started, completed or stopped. If not specified, then the request is repeated at regular intervals
	TrackerID     string // If a previus announce contained a tracker id, it should be here
	ConnectionID  []byte // Current connection ID to the tracker
	TransactionID []byte // Current transaction ID
}

// The TrackerResponse sent from the tracker
type TrackerResponse struct {
	FailureReason string `bencode:"failure reason,omitempty"`
	Interval      int    `bencode:"interval"`
	TrackerID     string `bencode:"tracker id"` // A string that the client shold send back on its next announcements. If absent and a previous announce sent an ID, use the old value
	Complete      int    `bencode:"complete"`   // Number of peers with the entire file
	Incomplete    int    `bencode:"incomplete"` // number of leechers; non-seeder peers
	Peers         string `bencode:"peers"`
	//Peers         []Peer `bencode:"peers"`
}

// A Peer is a participant in the swarm
type Peer struct {
	PeerID     string
	Address    string
	Bitfield   []int
	Interested int
	reqChannel chan []byte
	closeConn  chan int
}

// The Handshake is a required message and must be the first message transmitted by the client to a peer.
type Handshake struct {
	Pstrlen  byte     //string length of <pstr>
	Pstr     string   //string identifier of the protocol
	Reserved [8]byte  // Can be used to change behavior of the protocol
	InfoHash [20]byte // URLencoded 20-byte SHA1 hash of the value of the info key from the MetaInfo file, bencoded.
	PeerID   [20]byte // URLencoded 20-byte string used as a unique ID for the client, and generated by the client at startup.
}

// Piece represents the yet to be concacenated pieces that make up the file(s) being downloaded
type Piece struct {
	Index          int
	Hash           []byte
	Blocks         []Block
	Complete       bool
	AvailablePeers []*Peer
}

// A Block is a subset of a piece, and is what is actualy downloaded p2p before being assembled programmatically
type Block struct {
	Offset int
	Data   []byte
}
