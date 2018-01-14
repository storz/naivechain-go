package main

import (
	"flag"
	"log"

	"github.com/storz/naivechain-go"
)

func Main() error {
	var httpAddr string
	var peerAddr string
	flag.StringVar(&httpAddr, "http", ":3001", "http listen address")
	flag.StringVar(&peerAddr, "peer", ":6001", "peer address")
	flag.Parse()

	node, err := naivechain.New()
	if err != nil {
		return err
	}
	go node.HTTPServer().Run(httpAddr)
	go node.P2PServer().Run(peerAddr)

	select {}

	return nil
}

func main() {
	if err := Main(); err != nil {
		log.Fatal(err)
	}
}
