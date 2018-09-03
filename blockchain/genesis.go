package blockchain

import (
	"time"

	"github.com/ninjadotorg/cash-prototype/common"
	"github.com/ninjadotorg/cash-prototype/transaction"
)

type GenesisBlockGenerator struct {
}

func (self GenesisBlockGenerator) CalcMerkleRoot(txns []transaction.Transaction) common.Hash {
	if len(txns) == 0 {
		return common.Hash{}
	}

	utilTxns := make([]transaction.Transaction, 0, len(txns))
	for _, tx := range txns {
		utilTxns = append(utilTxns, tx)
	}
	merkles := Merkle{}.BuildMerkleTreeStore(utilTxns)
	return *merkles[len(merkles)-1]
}

func (self GenesisBlockGenerator) CreateGenesisBlock(time time.Time, nonce int, difficulty uint32, version int, genesisReward float64) *Block {
	genesisBlock := Block{}
	// update default genesis block
	genesisBlock.Header.Timestamp = time
	//genesisBlock.Header.PrevBlockHash = (&common.Hash{}).String()
	genesisBlock.Header.Nonce = nonce
	genesisBlock.Header.Difficulty = difficulty
	genesisBlock.Header.Version = version

	tx := transaction.Tx{
		Version: 1,
		TxIn: []transaction.TxIn{
			{
				Sequence:        0xffffffff,
				SignatureScript: []byte{},
				PreviousOutPoint: transaction.OutPoint{
					Hash: common.Hash{},
				},
			},
		},
		TxOut: []transaction.TxOut{
			{
				Value:     genesisReward,
				PkScript:  []byte(GENESIS_BLOCK_PUBKEY_ADDR),
				TxOutType: common.TxOutCoinType,
			},
		},
		Type: common.TxNormalType,
	}
	genesisBlock.Header.MerkleRoot = self.CalcMerkleRoot(genesisBlock.Transactions)
	genesisBlock.Transactions = append(genesisBlock.Transactions, &tx)
	return &genesisBlock
}

func (self GenesisBlockGenerator) CreateGenesisBlockPoSParallel(time time.Time, nonce int, difficulty uint32, version int, initialCoin float64, preSelectValidators []string) *Block {
	genesisBlock := Block{}
	// update default genesis block
	genesisBlock.Header.Timestamp = time
	//genesisBlock.Header.PrevBlockHash = (&common.Hash{}).String()
	genesisBlock.Header.Nonce = nonce
	genesisBlock.Header.Difficulty = difficulty
	genesisBlock.Header.Version = version
	genesisBlock.Header.NextCommittee = preSelectValidators
	tx := transaction.Tx{
		Version: 1,
		TxIn: []transaction.TxIn{
			{
				Sequence:        0xffffffff,
				SignatureScript: []byte{},
				PreviousOutPoint: transaction.OutPoint{
					Hash: common.Hash{},
				},
			},
		},
		TxOut: []transaction.TxOut{
			{
				Value:     initialCoin,
				PkScript:  []byte(GENESIS_BLOCK_PUBKEY_ADDR),
				TxOutType: common.TxOutCoinType,
			},
		},
		Type: common.TxNormalType,
	}
	genesisBlock.Header.MerkleRoot = self.CalcMerkleRoot(genesisBlock.Transactions)
	genesisBlock.Transactions = append(genesisBlock.Transactions, &tx)
	return &genesisBlock
}
