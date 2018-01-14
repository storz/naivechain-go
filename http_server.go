package naivechain

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"strings"

	"os"

	"github.com/pkg/errors"
)

type (
	HTTPServer struct {
		logger *log.Logger
		node   *Node
		p2ps   *P2PServer
	}
)

func (s *HTTPServer) Run(addr string) error {
	if s == nil || s.node == nil {
		return errors.New("server is not initialized")
	}
	http.HandleFunc("/blocks", s.blocks)
	http.HandleFunc("/mineBlock", s.mineBlock)
	http.HandleFunc("/peers", s.peers)
	http.HandleFunc("/addPeer", s.addPeer)

	if s.logger == nil {
		s.logger = log.New(os.Stdout, "[HTTP]", log.LstdFlags|log.Lmicroseconds)
	}

	var origin string
	if strings.HasPrefix(addr, ":") {
		origin = "http://localhost" + addr
	} else {
		origin = addr
	}
	s.logger.Printf("listening HTTP on: %s", origin)

	return http.ListenAndServe(addr, nil)
}

func (s *HTTPServer) blocks(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(s.node.Blockchain())
}

func (s *HTTPServer) mineBlock(w http.ResponseWriter, r *http.Request) {
	if s.p2ps == nil {
		return
	}

	var b Block
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		s.logger.Printf("failed to parse body: %v", err)
		fmt.Fprint(w, "internal error")
		return
	}

	newBlock := s.node.blockchain.GenerateNextBlock(b.Data)
	if err := s.node.blockchain.AddBlock(newBlock); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		s.logger.Printf("failed to add blcok: %v", err)
		fmt.Fprint(w, "internal error")
		return
	}

	s.p2ps.broadcastBlock(newBlock)
	s.logger.Printf("new block added: %v", newBlock)
}

func (s *HTTPServer) peers(w http.ResponseWriter, r *http.Request) {
	if s.p2ps == nil {
		return
	}

	ps := make([]string, 0, len(s.p2ps.conns))
	for conn := range s.p2ps.conns {
		ps = append(ps, conn.RemoteAddr().String())
	}
	json.NewEncoder(w).Encode(ps)
}

func (s *HTTPServer) addPeer(w http.ResponseWriter, r *http.Request) {
	var b bytes.Buffer
	if _, err := b.ReadFrom(r.Body); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		s.logger.Printf("failed to parse body: %v", err)
		fmt.Fprint(w, "internal error")
		return
	}
	if err := s.p2ps.connectToPeer(b.String()); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		s.logger.Printf("failed to connect to peer: %v", err)
		fmt.Fprint(w, "internal error")
		return
	}
}
