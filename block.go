package naivechain

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/pkg/errors"
)

type (
	Block struct {
		Index        int64     `json:"index"`
		PreviousHash string    `json:"previous_hash"`
		Timestamp    time.Time `json:"timestamp"`
		Data         string    `json:"data"`
		Hash         string    `json:"hash"`
	}
)

func GetGenesisBlock() Block {
	return Block{
		Index:        0,
		PreviousHash: "0",
		Timestamp:    time.Unix(1465154705, 0),
		Data:         "my genesis block!!",
		Hash:         "816534932c2b7154836da6afc367695e6337db8a921823784c14378abed4f7d7",
	}
}

func generateNextBlock(previousBlock Block, blockData string) Block {
	nextIndex := previousBlock.Index + 1
	nextTimestamp := time.Now()
	nextHash := hash(nextIndex, previousBlock.Hash, nextTimestamp, blockData)
	return Block{
		Index:        nextIndex,
		PreviousHash: previousBlock.Hash,
		Timestamp:    nextTimestamp,
		Data:         blockData,
		Hash:         nextHash,
	}
}

func hash(index int64, previousHash string, timestamp time.Time, data string) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%d%s%d%s", index, previousHash, timestamp.Unix(), data)))
	return hex.EncodeToString(sum[:])
}

func (b Block) hash() string {
	return hash(b.Index, b.PreviousHash, b.Timestamp, b.Data)
}

func IsValidNewBlock(newBlock, previousBlock Block) (bool, error) {
	if previousBlock.Index != newBlock.Index-1 {
		return false, errors.New("invalid index")
	}
	if previousBlock.Hash != newBlock.PreviousHash {
		return false, errors.New("invalid previous hash")
	}
	if previousBlock.hash() != newBlock.PreviousHash {
		return false, errors.New(fmt.Sprintf("invalid hash: calcurated=%s recorded=%s",
			previousBlock.hash(), newBlock.PreviousHash))
	}
	return true, nil
}
