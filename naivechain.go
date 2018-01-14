package naivechain

import (
	"github.com/pkg/errors"
	"golang.org/x/net/websocket"
)

type Node struct {
	blockchain *Chain
	httpServer *HTTPServer
	p2pServer  *P2PServer
}

type Client interface {
	Blockchain() *Chain
	HTTPServer() *HTTPServer
	P2PServer() *P2PServer
}

func New() (Client, error) {
	n := &Node{
		blockchain: &Chain{GetGenesisBlock()},
	}

	n.p2pServer = &P2PServer{
		node:  n,
		conns: make(map[*websocket.Conn]struct{}),
	}
	n.httpServer = &HTTPServer{
		node: n,
		p2ps: n.p2pServer,
	}

	return n, nil
}

func (n *Node) Blockchain() *Chain {
	return n.blockchain
}

func (n *Node) HTTPServer() *HTTPServer {
	return n.httpServer
}

func (n *Node) P2PServer() *P2PServer {
	return n.p2pServer
}

func (n *Node) replaceChain(newChain Chain) error {
	if _, err := newChain.IsValid(); err != nil {
		return errors.Wrap(err, "invalid chain")
	}
	if len(newChain) <= len(*n.blockchain) {
		return errors.New("invalid length")
	}

	n.blockchain = &newChain
	n.p2pServer.broadcastBlock(n.blockchain.LatestBlock())

	return nil
}
