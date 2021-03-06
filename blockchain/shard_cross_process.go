package blockchain

import (
	"errors"
	"math"

	"github.com/ninjadotorg/constant/common"
	"github.com/ninjadotorg/constant/metadata"
	"github.com/ninjadotorg/constant/privacy"
)

//Receive tx list from shard block body, produce merkle path of UTXO CrossShard List from specific shardID
func GetMerklePathCrossShard(txList []metadata.Transaction, shardID byte) (merklePathShard []common.Hash, merkleShardRoot common.Hash) {
	//calculate output coin hash for each shard
	outputCoinHash := getOutCoinHashEachShard(txList)
	// calculate merkel path for a shardID
	// step 1: calculate merkle data : 1 2 3 4 12 34 1234
	merkleData := outputCoinHash
	cursor := 0
	for {
		v1 := merkleData[cursor]
		v2 := merkleData[cursor+1]
		merkleData = append(merkleData, common.HashH(append(v1.GetBytes(), v2.GetBytes()...)))
		cursor += 2
		if cursor >= len(merkleData)-1 {
			break
		}
	}

	// step 2: get merkle path
	cursor = 0
	lastCursor := 0
	sid := int(shardID)
	i := sid
	for {
		if cursor >= len(merkleData)-2 {
			break
		}
		if i%2 == 0 {
			merklePathShard = append(merklePathShard, merkleData[cursor+i+1])
		} else {
			merklePathShard = append(merklePathShard, merkleData[cursor+i-1])
		}
		i = int(math.Floor(float64(i / 2)))

		if cursor == 0 {
			cursor += len(outputCoinHash)
		} else {
			tmp := cursor
			cursor += int(math.Floor(float64((cursor - lastCursor) / 2)))
			lastCursor = tmp
		}
	}
	merkleShardRoot = merkleData[len(merkleData)-1]
	return merklePathShard, merkleShardRoot
}

//Receive a cross shard block and merkle path, verify whether the UTXO list is valid or not
func VerifyCrossShardBlockUTXO(block *CrossShardBlock, merklePathShard []common.Hash) bool {
	outCoins := block.CrossOutputCoin
	tmpByte := []byte{}
	for _, coin := range outCoins {
		tmpByte = append(tmpByte, coin.Bytes()...)
	}
	finalHash := common.HashH(tmpByte)
	for _, hash := range merklePathShard {
		finalHash = common.HashH(append(finalHash.GetBytes(), hash.GetBytes()...))
	}
	return VerifyMerkleTree(finalHash, merklePathShard, block.Header.ShardTxRoot)
}

func VerifyMerkleTree(finalHash common.Hash, merklePath []common.Hash, merkleRoot common.Hash) bool {
	for _, hashPath := range merklePath {
		finalHash = common.HashH(append(finalHash.GetBytes(), hashPath.GetBytes()...))
	}
	if finalHash != merkleRoot {
		return false
	} else {
		return true
	}
}

// helper function to group OutputCoin into shard and get the hash of each group
func getOutCoinHashEachShard(txList []metadata.Transaction) []common.Hash {
	// group transaction by shardID
	outCoinEachShard := make([][]*privacy.OutputCoin, common.SHARD_NUMBER)
	for _, tx := range txList {
		for _, outCoin := range tx.GetProof().OutputCoins {
			lastByte := outCoin.CoinDetails.GetPubKeyLastByte()
			shardID := common.GetShardIDFromLastByte(lastByte)
			outCoinEachShard[shardID] = append(outCoinEachShard[shardID], outCoin)
		}
	}

	//calcualte hash for each shard
	outputCoinHash := make([]common.Hash, common.SHARD_NUMBER)
	for i := 0; i < common.SHARD_NUMBER; i++ {
		if len(outCoinEachShard[i]) == 0 {
			outputCoinHash[i] = common.HashH([]byte(""))
		} else {
			tmpByte := []byte{}
			for _, coin := range outCoinEachShard[i] {
				tmpByte = append(tmpByte, coin.Bytes()...)
			}
			outputCoinHash[i] = common.HashH(tmpByte)
		}
	}
	return outputCoinHash
}

// helper function to get the hash of OutputCoins (send to a shard) from list of transaction
func getOutCoinCrossShard(txList []metadata.Transaction, shardID byte) []privacy.OutputCoin {
	coinList := []privacy.OutputCoin{}
	for _, tx := range txList {
		for _, outCoin := range tx.GetProof().OutputCoins {
			lastByte := outCoin.CoinDetails.GetPubKeyLastByte()
			if lastByte == shardID {
				coinList = append(coinList, *outCoin)
			}
		}
	}
	return coinList
}

/*
	Verify CrossShard Block
	- Agg Signature
	- MerklePath
*/
func (self *CrossShardBlock) VerifyCrossShardBlock(committees []string) error {
	if err := ValidateAggSignature(self.ValidatorsIdx, committees, self.AggregatedSig, self.R, self.Hash()); err != nil {
		return NewBlockChainError(SignatureError, err)
	}
	if err := VerifyCrossShardBlockUTXO(self, self.MerklePathShard); err == false {
		return NewBlockChainError(HashError, errors.New("Verify Merkle Path Shard"))
	}
	return nil
}
