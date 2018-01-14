package naivechain

import "github.com/pkg/errors"

type (
	Chain []Block
)

func addBlock(c *Chain, b Block) error {
	if _, err := IsValidNewBlock(b, c.LatestBlock()); err != nil {
		return errors.Wrap(err, "failed to add block")
	}
	*c = append(*c, b)
	return nil
}

func (c *Chain) LatestBlock() Block {
	return (*c)[len(*c)-1]
}

func (c *Chain) GenerateNextBlock(blockData string) Block {
	return generateNextBlock(c.LatestBlock(), blockData)
}

func (c *Chain) AddBlock(b Block) error {
	return addBlock(c, b)
}

func (c Chain) IsValid() (bool, error) {
	if len(c) == 0 {
		return false, errors.New("length is zero")
	}
	if c[0].hash() != GetGenesisBlock().hash() {
		return false, errors.New("genesis block is mismatched")
	}
	for i := 1; i < len(c); i++ {
		if _, err := IsValidNewBlock(c[i], c[i-1]); err != nil {
			return false, errors.Wrapf(err, "invalid block between: %d->%d", i-1, i)
		}
	}
	return true, nil
}
