package client

// Will handle exchanging of messages between peers

import (
	"bufio"
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"net"
	"time"
)

var (
	amChoking      []byte
	amInterested   []byte // unchoke message
	peerChoking    []byte // uninterested message
	peerInterested []byte
)

const (
	keepAlive = byte(0)
)

func init() {
	amChoking = func() []byte {
		message := make([]byte, 5)
		binary.BigEndian.PutUint32(message[:4], uint32(1))
		copy(message[4:], string(0))
		return message
	}()
	amInterested = func() []byte {
		message := make([]byte, 5)
		binary.BigEndian.PutUint32(message[:4], uint32(1))
		copy(message[4:], string(1))
		return message
	}()
	peerChoking = func() []byte {
		message := make([]byte, 5)
		binary.BigEndian.PutUint32(message[:4], uint32(1))
		copy(message[4:], string(2))
		return message
	}()
	peerInterested = func() []byte {
		message := make([]byte, 5)
		binary.BigEndian.PutUint32(message[:4], uint32(1))
		copy(message[4:], string(3))
		return message
	}()
}

// Handshake generate connection, send message, get reply, check peer_id
func (peer *Peer) Handshake(infoHash []byte) *net.TCPConn {
	raddr, err := net.ResolveTCPAddr("tcp", peer.Address)
	fmt.Println(raddr.String())

	if err != nil {
		fmt.Printf("Unable to resolve peer IP address: %s \n", err.Error())
		return nil
	}
	conn, err := net.DialTCP("tcp", nil, raddr)
	if err != nil {
		fmt.Printf("Unable to connect with provided peer address (%s): %s \n", raddr.String(), err.Error())
		return nil
	}

	fmt.Println("Peer connection open")

	conn.SetDeadline(time.Now().Add(time.Second * time.Duration(20)))

	message := make([]byte, 68)

	copy(message[0:], string(19))
	copy(message[1:20], "BitTorrent protocol")
	binary.BigEndian.PutUint64(message[20:28], uint64(0))
	copy(message[28:48], infoHash)
	copy(message[48:], MyPeerID)

	fmt.Println(message[67:])

	nWritten, err := conn.Write(message[:])
	if err != nil {
		fmt.Printf("Unable to write to TCP connection: %s \n", err.Error())
		return nil
	}

	fmt.Printf("%v bytes written to address: %v \n", nWritten, conn.RemoteAddr())

	for {
		response, err := bufio.NewReader(conn).ReadString('0')
		if err != nil {
			fmt.Printf("Unable to read from tcp connection: %s", err.Error())
		}
		if len(response) > 0 {
			fmt.Println(response)
			break
		}
	}
	return conn
}

// HandlePeer will loop until peer closes connection, processing and routing messages sent through said connection
func (peer *Peer) HandlePeer(conn *net.TCPConn, torrent *Torrent) {

	defer conn.Close()
	// create a bunch of channels for message signaling
	block := make(chan []byte)
	choking := make(chan []byte)
	interested := make(chan []byte)
	bitfield := make(chan []byte)
	request := make(chan []byte)
	cancel := make(chan []byte)
	have := make(chan []byte)

	channelMap := make(map[string](chan []byte))
	channelMap["block"] = block
	channelMap["choking"] = choking
	channelMap["interested"] = interested
	channelMap["bitfield"] = bitfield
	channelMap["request"] = request
	channelMap["have"] = request
	channelMap["cancel"] = cancel

	go listen(conn, channelMap)

	/* Here's where we route the received messages to their needed functions.
	   For example, if we confirm that we received a block message using the
	   evaluateMessage(), it will route the message to a processing helper function
	   to append the block at its index. */
	for {
		select {
		case <-peer.closeConn:
			fmt.Println("Peer connection closed.")
			return
		// The peer.reqChannel channel receives requests from the client on what it wants from a particular peer.
		case request := <-peer.reqChannel:
			sendMessage(request, conn)
			return
		case blkMsg := <-block:
			go torrent.processBlock(blkMsg)
			continue
		case reqMsg := <-request:
			go sendMessage(torrent.processRequest(reqMsg), conn)
			continue
		case haveMsg := <-have:
			go peer.processHave(haveMsg)
			continue
		case chkMsg := <-choking:
			if chkMsg[0] == 0 {
				// peer is now choking us
				peer.waitForUnchoke(conn, choking)
				continue
			}
		case interestedMsg := <-interested:
			if interestedMsg[0] == 2 {
				peer.Interested = 0
				continue
			}
			peer.Interested = 3
			continue
		case bfieldMsg := <-bitfield:
			go peer.processBitfield(bfieldMsg)
			continue
		case cancelMessage := <-cancel:
			go torrent.processCancel(cancelMessage)
			continue
		}
	}

}

// sendMessage takes a func that returns a built message and a connection and sends that message over the connection
func sendMessage(msg []byte, conn *net.TCPConn) {
	_, err := conn.Write(msg)
	if err != nil {
		fmt.Printf("Could not send given message to %v: %s \n", conn.RemoteAddr(), err.Error())
	}
}

func listen(conn *net.TCPConn, signal map[string](chan []byte)) {
	response := make([]byte, 8192)
	for {
		nRead, err := conn.Read(response)
		if err != nil {
			fmt.Printf("Cannot read from connection: %s \n", err.Error())
		}

		if nRead > 0 {
			go evaluateMessage(signal, response[:nRead])

			response = make([]byte, 8192)
		}
	}
}

func evaluateMessage(signal map[string](chan []byte), msg []byte) {
	if len(msg) == 4 {
		// It's a keep alive. By being received it's already served it's purpose
		return
	}
	/* The range depicted should be an int32 representing the id of the message.
	   The core part of our evaluation logic is checking the id and signaling to
	   the appropriate channel with the relevant information.
	*/
	id := msg[4]

	if id == 0 {
		signal["choking"] <- []byte{0}
		return
	} else if id == 1 {
		signal["choking"] <- []byte{1}
		return
	} else if id == 2 {
		signal["interested"] <- []byte{2}
		return
	} else if id == 3 {
		signal["interested"] <- []byte{3}
		return
	} else if id == 4 {
		signal["have"] <- msg
		return
	} else if id == 5 {
		signal["bitfield"] <- msg
		return
	} else if id == 6 {
		signal["request"] <- msg
		return
	} else if id == 7 {
		signal["block"] <- msg
		return
	} else if id == 8 {
		signal["cancel"] <- msg
		return
	}
}

func (torrent *Torrent) processBlock(blkMsg []byte) {
	/*	piece: <len=0009+X><id=7><index><begin><block>

		The piece message is variable length, where X is the length of the block. The payload contains the following information:

			index: integer specifying the zero-based piece index
			begin: integer specifying the zero-based byte offset within the piece
			block: block of data, which is a subset of the piece specified by index.
	*/
	length := int32(binary.BigEndian.Uint32(blkMsg[0:4]))
	index := int32(binary.BigEndian.Uint32(blkMsg[5:9]))
	begin := int32(binary.BigEndian.Uint32(blkMsg[9:13]))
	blk := blkMsg[13:length]

	/*	Iterate through the torrents pieces and find the one at the
		given index. Once found, iterate through the blocks of that
		piece until you find one with an offset that matches the given
		'begin' value. Assign the blk data to that block. Every time the
		function is called, compare the complete hash of the piece given
		by the tracker and the current has of every block. If they equate,
		say the piece is complete.
	*/
	for _, piece := range torrent.Pieces {
		if piece.Index == int(index) {
			var blocks []byte
			for _, block := range piece.Blocks {
				copy(blocks[(len(blocks)+1):], block.Data)
				if block.Offset == int(begin) {
					block.Data = blk
				}
			}
			hash := sha1.New()
			hash.Write(blocks)
			if bytes.Compare(hash.Sum(nil), piece.Hash) == 0 {
				var data []byte
				for _, blk := range piece.Blocks {
					copy(data[len(data):], blk.Data)
				}
				torrent.writeToFile(data, torrent.Data.Info.PieceLength*piece.Index)
				piece.Complete = true
			}
			return
		}
	}
	fmt.Println("Block received not found compatible with any piece.")
	return
}

// Assume a request has been sent only if the peer knows you have the piece (which should be the case in every situation.)
func (torrent *Torrent) processRequest(reqMsg []byte) []byte {
	/*	request: <len=0013><id=6><index><begin><length>

		The request message is fixed length, and is used to request a block. The payload contains the following information:

			index: integer specifying the zero-based piece index
			begin: integer specifying the zero-based byte offset within the piece
			length: integer specifying the requested length.
	*/
	index := int32(binary.BigEndian.Uint32(reqMsg[5:9]))
	begin := int32(binary.BigEndian.Uint32(reqMsg[9:13]))
	length := int32(binary.BigEndian.Uint32(reqMsg[13:17]))

	var blk []byte
	for _, piece := range torrent.Pieces {
		if piece.Index == int(index) {
			for _, block := range piece.Blocks {
				if block.Offset == int(begin) {
					blk = block.Data[:int(length)]
					break
				}
			}

		}
	}
	var reply []byte

	binary.BigEndian.PutUint32(reply[0:], uint32(9+len(blk)))
	reply[4] = 7
	copy(reply[5:9], reqMsg[5:9])
	copy(reply[9:13], reqMsg[9:13])
	copy(reply[13:17], reqMsg[13:17])
	copy(reply[17:], blk)

	return reply

}

func (peer *Peer) processHave(haveMsg []byte) {
	/*	have: <len=0005><id=4><piece index>

		The have message is fixed length. The payload is the zero-based index of a piece that has just been
		successfully downloaded and verified via the hash.
	*/

	index := int32(binary.BigEndian.Uint32(haveMsg[5:9]))
	// Update the specified index in the peers bitfield to say they have the piece.
	peer.Bitfield[index] = 1

}

func (peer *Peer) processBitfield(bfieldMsg []byte) {
	/*	bitfield: <len=0001+X><id=5><bitfield>

		The bitfield message may only be sent immediately after the handshaking sequence is completed, and before any
		other messages are sent. It is optional, and need not be sent if a client has no pieces.

		The bitfield message is variable length, where X is the length of the bitfield. The payload is a bitfield
		representing the pieces that have been successfully downloaded. The high bit in the first byte corresponds
		to piece index 0. Bits that are cleared indicated a missing piece, and set bits indicate a valid and available
		piece. Spare bits at the end are set to zero.

		Some clients (Deluge for example) send bitfield with missing pieces even if it has all data. Then it sends rest
		of pieces as have messages. They are saying this helps against ISP filtering of BitTorrent protocol. It is called
		lazy bitfield.
	*/

	peer.Bitfield = getBitsFromByteSlice(bfieldMsg[5:])

}

func (torrent *Torrent) processCancel(cancelMessage []byte) {
	/*	cancel: <len=0013><id=8><index><begin><length>

		The cancel message is fixed length, and is used to cancel block requests. The payload is identical to that of the
		"request" message. It is typically used during "End Game" (see the Algorithms section below).
	*/
	return
}

// This whole function might be a waste of time... review.
func (peer *Peer) waitForUnchoke(conn *net.TCPConn, choking chan []byte) {
	ticker := time.NewTicker(115 * time.Second)
	defer ticker.Stop()
Loop:
	for {
		// peer is currently choking us. Loop until it's ready, sending occassional keep alives.
		select {
		case <-ticker.C:
			sendMessage([]byte{keepAlive}, conn)
		case chkMsg := <-choking:
			if chkMsg[0] == 1 {
				// unchoked. Start processing again.
				break Loop
			}
		default:
			continue
		}
	}

	return
}

func createRequest(piece *Piece) []byte {
	// request: <len=0013><id=6><index><begin><length>
	request := make([]byte, 17)
	binary.BigEndian.PutUint32(request[:4], uint32(13))
	copy(request[4:], string(6))
	binary.BigEndian.PutUint32(request[5:9], uint32(piece.Index))

	for _, blk := range piece.Blocks {
		if len(blk.Data) == 0 {
			binary.BigEndian.PutUint32(request[9:13], uint32(blk.Offset))
			break
		}
	}

	binary.BigEndian.PutUint32(request[13:17], uint32(16384))

	return request
}
