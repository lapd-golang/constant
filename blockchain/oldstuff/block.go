package blockchain

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"github.com/ninjadotorg/constant/common"
	"github.com/ninjadotorg/constant/metadata"
	"github.com/ninjadotorg/constant/transaction"
)

/*
Block is struct present every block in blockchain
block contains many types of transaction
- normal tx:
- action tx:

*/
type Block struct {
	Header           BlockHeader
	Transactions     []metadata.Transaction
	BlockProducer    string // in base58check.encode
	BlockProducerSig string

	blockHash *common.Hash
}

/*
Customize UnmarshalJSON to parse list TxNormal
because we have many types of block, so we can need to customize data from marshal from json string to build a block
*/
func (self *Block) UnmarshalJSON(data []byte) error {
	type Alias Block
	temp := &struct {
		Transactions []map[string]interface{}
		*Alias
	}{
		Alias: (*Alias)(self),
	}

	err := json.Unmarshal(data, &temp)
	if err != nil {
		return NewBlockChainError(UnmashallJsonBlockError, err)
	}

	// process tx from tx interface of temp
	for _, txTemp := range temp.Transactions {
		txTempJson, _ := json.MarshalIndent(txTemp, "", "\t")
		Logger.log.Debugf("Tx json data: ", string(txTempJson))

		var tx metadata.Transaction
		var parseErr error
		switch txTemp["Type"].(string) {
		case common.TxNormalType:
			{
				tx = &transaction.Tx{}
				parseErr = json.Unmarshal(txTempJson, &tx)
			}
		case common.TxSalaryType:
			{
				tx = &transaction.Tx{}
				parseErr = json.Unmarshal(txTempJson, &tx)
			}
		case common.TxCustomTokenType:
			{
				tx = &transaction.TxCustomToken{}
				parseErr = json.Unmarshal(txTempJson, &tx)
			}
		case common.TxCustomTokenPrivacyType:
			{
				tx = &transaction.TxCustomTokenPrivacy{}
				parseErr = json.Unmarshal(txTempJson, &tx)
			}
		default:
			{
				return NewBlockChainError(UnmashallJsonBlockError, errors.New("Can not parse a wrong tx"))
			}
		}

		if parseErr != nil {
			return NewBlockChainError(UnmashallJsonBlockError, parseErr)
		}
		/*meta, parseErr := metadata.ParseMetadata(txTemp["Metadata"])
		if parseErr != nil {
			return NewBlockChainError(UnmashallJsonBlockError, parseErr)
		}
		tx.SetMetadata(meta)*/
		self.Transactions = append(self.Transactions, tx)
	}

	self.Header = temp.Alias.Header
	return nil
}

/*
AddTransaction adds a new transaction into block
*/
// #1 - tx
func (self *Block) AddTransaction(tx metadata.Transaction) error {
	if self.Transactions == nil {
		return NewBlockChainError(UnExpectedError, errors.New("Not init tx arrays"))
	}
	self.Transactions = append(self.Transactions, tx)
	return nil
}

/*
Hash creates a hash from block data
*/
func (self Block) Hash() *common.Hash {
	if self.blockHash != nil {
		return self.blockHash
	}

	record := ""

	// add data from header
	record += strconv.FormatInt(self.Header.Timestamp, 10) +
		string(self.Header.ShardID) +
		self.Header.MerkleRoot.String() +
		//self.Header.MerkleRootCommitments.String() +
		self.Header.PrevBlockHash.String() +
		strconv.Itoa(int(self.Header.SalaryFund)) +
		strconv.Itoa(int(self.Header.GOVConstitution.GOVParams.SalaryPerTx)) +
		strconv.Itoa(int(self.Header.GOVConstitution.GOVParams.BasicSalary)) +
		strings.Join(self.Header.Committee, ",")

	// add data from body
	record += strconv.Itoa(self.Header.Version) +
		self.BlockProducer +
		self.BlockProducerSig +
		strconv.Itoa(len(self.Transactions)) +
		strconv.Itoa(int(self.Header.Height))

	// add data from tx
	for _, tx := range self.Transactions {
		record += tx.Hash().String()
	}

	hash := common.DoubleHashH([]byte(record))
	self.blockHash = &hash
	return self.blockHash
}

func (block *Block) updateDCBConstitution(tx metadata.Transaction, blockgen *BlkTmplGenerator) error {
	metadataAcceptDCBProposal := tx.GetMetadata().(*metadata.AcceptDCBProposalMetadata)
	_, _, _, getTx, err := blockgen.chain.GetTransactionByHash(&metadataAcceptDCBProposal.DCBProposalTXID)
	DCBProposal := getTx.GetMetadata().(*metadata.SubmitDCBProposalMetadata)
	previousConstitutionIndex := blockgen.chain.GetConstitutionIndex(DCBConstitutionHelper{})
	newConstitutionIndex := previousConstitutionIndex + 1
	if err != nil {
		return err
	}
	constitutionInfo := NewConstitutionInfo(
		newConstitutionIndex,
		uint64(block.Header.Height),
		DCBProposal.ExecuteDuration,
		DCBProposal.Explanation,
		*metadataAcceptDCBProposal.Hash(),
	)
	block.Header.DCBConstitution = *NewDCBConstitution(constitutionInfo, GetOracleDCBNationalWelfare(), &DCBProposal.DCBParams)
	return nil
}

func (block *Block) updateGOVConstitution(tx metadata.Transaction, blockgen *BlkTmplGenerator) error {
	metadataAcceptGOVProposal := tx.GetMetadata().(*metadata.AcceptGOVProposalMetadata)
	_, _, _, getTx, err := blockgen.chain.GetTransactionByHash(&metadataAcceptGOVProposal.GOVProposalTXID)
	GOVProposal := getTx.GetMetadata().(*metadata.SubmitGOVProposalMetadata)
	previousConstitutionIndex := blockgen.chain.GetConstitutionIndex(GOVConstitutionHelper{})
	newConstitutionIndex := previousConstitutionIndex + 1
	if err != nil {
		return err
	}
	constitutionInfo := NewConstitutionInfo(
		newConstitutionIndex,
		uint64(block.Header.Height),
		GOVProposal.ExecuteDuration,
		GOVProposal.Explanation,
		*metadataAcceptGOVProposal.Hash(),
	)
	block.Header.GOVConstitution = *NewGOVConstitution(constitutionInfo, GetOracleGOVNationalWelfare(), &GOVProposal.GOVParams)
	return nil
}

func (block *Block) updateBlock(
	blockgen *BlkTmplGenerator,
	txGroups *txGroups,
	accumulativeValues *accumulativeValues,
	updatedOracleValues map[string]uint64,
) error {
	if block.Header.GOVConstitution.GOVParams.SellingBonds != nil {
		block.Header.GOVConstitution.GOVParams.SellingBonds.BondsToSell -= accumulativeValues.bondsSold
	}
	if block.Header.GOVConstitution.GOVParams.SellingGOVTokens != nil {
		block.Header.GOVConstitution.GOVParams.SellingGOVTokens.GOVTokensToSell -= accumulativeValues.govTokensSold
	}
	if block.Header.DCBConstitution.DCBParams.SaleDCBTokensByUSDData != nil {
		block.Header.DCBConstitution.DCBParams.SaleDCBTokensByUSDData.Amount -= accumulativeValues.dcbTokensSold
	}

	blockgen.updateOracleValues(block, updatedOracleValues)
	err := blockgen.updateOracleBoard(block, txGroups.updatingOracleBoardTxs)
	if err != nil {
		Logger.log.Error(err)
		return err
	}

	for _, tx := range txGroups.txsToAdd {
		if err := block.AddTransaction(tx); err != nil {
			panic("add transaction failed")
			return err
		}
		// Handle if this transaction change something in block header or database
		if tx.GetMetadataType() == metadata.AcceptDCBProposalMeta {
			block.updateDCBConstitution(tx, blockgen)
		}
		if tx.GetMetadataType() == metadata.AcceptGOVProposalMeta {
			block.updateGOVConstitution(tx, blockgen)
		}
		if tx.GetMetadataType() == metadata.AcceptDCBBoardMeta {
			block.UpdateDCBBoard(tx)
		}
		if tx.GetMetadataType() == metadata.AcceptGOVBoardMeta {
			block.UpdateGOVBoard(tx)
		}
		if tx.GetMetadataType() == metadata.RewardDCBProposalSubmitterMeta {
			block.UpdateDCBFund(tx)
		}
		if tx.GetMetadataType() == metadata.RewardGOVProposalSubmitterMeta {
			block.UpdateGOVFund(tx)
		}
	}

	// register multisigs addresses
	err = blockgen.registerMultiSigsAddresses(txGroups.multiSigsRegistrationTxs)
	if err != nil {
		Logger.log.Error(err)
		return err
	}
	return nil
}
