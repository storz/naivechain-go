package naivechain

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/net/websocket"
)

const (
	MessageTypeQueryLatest        MessageType = 1
	MessageTypeQueryAll                       = 2
	MessageTypeResponseBlockchain             = 3
)

type (
	P2PServer struct {
		logger *log.Logger
		origin string
		node   *Node
		peers  []string
		conns  map[*websocket.Conn]struct{}
	}

	MessageType int
	Message     struct {
		MessageType MessageType `json:"message_type"`
		Blockchain  *Chain      `json:"blockchain,omitempty"`
	}
)

func (s *P2PServer) Run(addr string) error {
	if s == nil || s.node == nil {
		return errors.New("server is not initialized")
	}
	if s.logger == nil {
		s.logger = log.New(os.Stdout, "[P2P]", log.LstdFlags|log.Lmicroseconds)
	}

	if s.conns == nil {
		s.conns = make(map[*websocket.Conn]struct{})
	}

	if strings.HasPrefix(addr, ":") {
		s.origin = "http://localhost" + addr
	} else {
		s.origin = addr
	}

	http.Handle("/", websocket.Handler(s.initConnection))

	s.logger.Printf("listening websocket p2p on: %s", s.origin)
	return http.ListenAndServe(addr, nil)
}

func (s *P2PServer) initConnection(ws *websocket.Conn) {
	s.conns[ws] = struct{}{}

	// on error or close
	defer func() {
		delete(s.conns, ws)

		if err := ws.Close(); err != nil {
			s.logger.Printf("failed to close connection: %v", err)
			return
		}

		s.logger.Printf("connection closed: %s", ws.RemoteAddr().String())
	}()

	s.write(ws, Message{
		MessageType: MessageTypeQueryLatest,
	})
	s.listen(ws)
}

func (s *P2PServer) listen(conn *websocket.Conn) {
	for {
		var m Message
		if err := websocket.JSON.Receive(conn, &m); err != nil {
			if err != io.EOF {
				s.logger.Printf("receive failed: %v", err)
			}
			s.logger.Printf("failed to parse websocket message: %v", err)
			break
		}

		s.logger.Printf("received message: %s", m.toJSON())
		switch m.MessageType {
		case MessageTypeQueryLatest:
			s.writeBlock(conn, s.node.Blockchain().LatestBlock())
		case MessageTypeQueryAll:
			s.writeChain(conn, s.node.Blockchain())
		case MessageTypeResponseBlockchain:
			s.handleResponseBlockchain(m)
		}
	}
}

func (s *P2PServer) handleResponseBlockchain(m Message) {
	receivedBlocks := *m.Blockchain
	sort.Slice(receivedBlocks, func(i, j int) bool { return receivedBlocks[i].Index < receivedBlocks[j].Index })
	latestBlockReceived := receivedBlocks.LatestBlock()
	latestBlockHeld := s.node.Blockchain().LatestBlock()

	// Validate
	if latestBlockReceived.Index <= latestBlockHeld.Index {
		s.logger.Println("received blockchain is not longer than current blockchain")
		return
	}

	s.logger.Printf("blockchain possibly behind. We got: %d, Peer got: %d", latestBlockHeld.Index, latestBlockReceived.Index)

	switch {
	case latestBlockReceived.Hash == latestBlockReceived.PreviousHash:
		s.logger.Println("we can append the received block to our chain")
		s.node.Blockchain().AddBlock(latestBlockReceived)
		s.broadcastBlock(latestBlockReceived)
	case len(receivedBlocks) == 1:
		s.logger.Println("we have to query the chain from our peer")
		s.broadcast(Message{
			MessageType: MessageTypeQueryAll,
		})
	default:
		s.logger.Println("received blockchain is longer than current blockchain")
		s.node.replaceChain(receivedBlocks)
	}
}

func (s *P2PServer) write(target *websocket.Conn, msg Message) {
	s.logger.Printf("write %s to %s", msg.toJSON(), target.RemoteAddr())
	if err := websocket.JSON.Send(target, msg); err != nil {
		s.logger.Printf("failed to write: %v", err)
	}
}

func (s *P2PServer) writeChain(target *websocket.Conn, c *Chain) {
	s.write(target, Message{
		MessageType: MessageTypeResponseBlockchain,
		Blockchain:  c,
	})
}

func (s *P2PServer) writeBlock(target *websocket.Conn, b Block) {
	c := Chain([]Block{b})
	s.writeChain(target, &c)
}

func (s *P2PServer) broadcast(msg Message) {
	for ws := range s.conns {
		s.write(ws, msg)
	}
}

func (s *P2PServer) broadcastChain(c *Chain) {
	s.broadcast(Message{
		MessageType: MessageTypeResponseBlockchain,
		Blockchain:  c,
	})
}

func (s *P2PServer) broadcastBlock(b Block) {
	c := Chain([]Block{b})
	s.broadcastChain(&c)
}

func (s *P2PServer) connectToPeer(target string) error {
	s.logger.Printf("connect to %s", target)
	ws, err := websocket.Dial(target, "", s.origin)
	if err != nil {
		return errors.Wrap(err, "failed to dial")
	}
	go s.initConnection(ws)
	return nil
}

func (s *P2PServer) connectToPeers(peers []string) {
	for _, peer := range peers {
		s.connectToPeer(peer)
	}
}

func (m *Message) toJSON() string {
	var b bytes.Buffer
	if err := json.NewEncoder(&b).Encode(m); err != nil {
		return ""
	}
	return strings.TrimSpace(b.String())
}
