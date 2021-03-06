package blockchain

import (
	"github.com/ninjadotorg/constant/blockchain/params"
	"github.com/ninjadotorg/constant/common"
	"github.com/ninjadotorg/constant/database"
	"github.com/ninjadotorg/constant/database/lvdb"
	"github.com/ninjadotorg/constant/metadata"
	"github.com/ninjadotorg/constant/privacy"
	"github.com/ninjadotorg/constant/transaction"
)

type ConstitutionInfo struct {
	ConstitutionIndex  uint32
	StartedBlockHeight uint64
	ExecuteDuration    uint64
	Explanation        string
	AcceptProposalTXID common.Hash
}

func NewConstitutionInfo(constitutionIndex uint32, startedBlockHeight uint64, executeDuration uint64, explanation string, proposalTXID common.Hash) *ConstitutionInfo {
	return &ConstitutionInfo{
		ConstitutionIndex:  constitutionIndex,
		StartedBlockHeight: startedBlockHeight,
		ExecuteDuration:    executeDuration,
		Explanation:        explanation,
		AcceptProposalTXID: proposalTXID,
	}
}

func (constitutionInfo ConstitutionInfo) GetConstitutionIndex() uint32 {
	return constitutionInfo.ConstitutionIndex
}

type GOVConstitution struct {
	ConstitutionInfo
	CurrentGOVNationalWelfare int32
	GOVParams                 params.GOVParams
}

func NewGOVConstitution(constitutionInfo *ConstitutionInfo, currentGOVNationalWelfare int32, GOVParams *params.GOVParams) *GOVConstitution {
	return &GOVConstitution{
		ConstitutionInfo:          *constitutionInfo,
		CurrentGOVNationalWelfare: currentGOVNationalWelfare,
		GOVParams:                 *GOVParams,
	}
}

func (dcbConstitution DCBConstitution) GetEndedBlockHeight() uint64 {
	return dcbConstitution.StartedBlockHeight + dcbConstitution.ExecuteDuration
}

func (govConstitution GOVConstitution) GetEndedBlockHeight() uint64 {
	return govConstitution.StartedBlockHeight + govConstitution.ExecuteDuration
}

type DCBConstitution struct {
	ConstitutionInfo
	CurrentDCBNationalWelfare int32
	DCBParams                 params.DCBParams
}

func NewDCBConstitution(constitutionInfo *ConstitutionInfo, currentDCBNationalWelfare int32, DCBParams *params.DCBParams) *DCBConstitution {
	return &DCBConstitution{
		ConstitutionInfo:          *constitutionInfo,
		CurrentDCBNationalWelfare: currentDCBNationalWelfare,
		DCBParams:                 *DCBParams,
	}
}

type DCBConstitutionHelper struct{}
type GOVConstitutionHelper struct{}

func (helper DCBConstitutionHelper) GetConstitutionEndedBlockHeight(blockgen *BlkTmplGenerator, shardID byte) uint64 {
	// BestBlock := blockgen.chain.BestState[shardID].BestBlock
	// lastDCBConstitution := BestBlock.Header.DCBConstitution
	// return lastDCBConstitution.StartedBlockHeight + lastDCBConstitution.ExecuteDuration
	return 0
}

func (helper GOVConstitutionHelper) GetConstitutionEndedBlockHeight(blockgen *BlkTmplGenerator, shardID byte) uint64 {
	// BestBlock := blockgen.chain.BestState[shardID].BestBlock
	// lastGOVConstitution := BestBlock.Header.GOVConstitution
	// return lastGOVConstitution.StartedBlockHeight + lastGOVConstitution.ExecuteDuration
	return 0
}

func (helper DCBConstitutionHelper) GetStartedNormalVote(blockgen *BlkTmplGenerator, shardID byte) uint64 {
	// BestBlock := blockgen.chain.BestState[shardID].BestBlock
	// lastDCBConstitution := BestBlock.Header.DCBConstitution
	// return uint64(lastDCBConstitution.StartedBlockHeight) - uint64(common.EncryptionOnePhraseDuration)
	return 0
}

func (helper DCBConstitutionHelper) CheckSubmitProposalType(tx metadata.Transaction) bool {
	return tx.GetMetadataType() == metadata.SubmitDCBProposalMeta
}

func (helper DCBConstitutionHelper) GetAmountVoteTokenOfTx(tx metadata.Transaction) uint64 {
	return tx.(*transaction.TxCustomToken).GetAmountOfVote()
}

func (helper GOVConstitutionHelper) GetStartedNormalVote(blockgen *BlkTmplGenerator, shardID byte) uint64 {
	// BestBlock := blockgen.chain.BestState[shardID].BestBlock
	// lastGOVConstitution := BestBlock.Header.GOVConstitution
	// return uint64(lastGOVConstitution.StartedBlockHeight) - uint64(common.EncryptionOnePhraseDuration)
	return 0
}

func (helper GOVConstitutionHelper) CheckSubmitProposalType(tx metadata.Transaction) bool {
	return tx.GetMetadataType() == metadata.SubmitGOVProposalMeta
}

func (helper GOVConstitutionHelper) GetAmountVoteTokenOfTx(tx metadata.Transaction) uint64 {
	return tx.(*transaction.TxCustomToken).GetAmountOfVote()
}

func (helper DCBConstitutionHelper) TxAcceptProposal(
	txId *common.Hash,
	voter metadata.Voter,
	minerPrivateKey *privacy.SpendingKey,
	db database.DatabaseInterface,
) metadata.Transaction {
	meta := metadata.NewAcceptDCBProposalMetadata(*txId, voter)
	acceptTx := transaction.NewEmptyTx(minerPrivateKey, db, meta)
	return acceptTx
}

func (helper GOVConstitutionHelper) TxAcceptProposal(
	txId *common.Hash,
	voter metadata.Voter,
	minerPrivateKey *privacy.SpendingKey,
	db database.DatabaseInterface,
) metadata.Transaction {
	meta := metadata.NewAcceptGOVProposalMetadata(*txId, voter)
	acceptTx := transaction.NewEmptyTx(minerPrivateKey, db, meta)
	return acceptTx
}

func (helper DCBConstitutionHelper) GetBoardType() string {
	return "dcb"
}

func (helper GOVConstitutionHelper) GetBoardType() string {
	return "gov"
}

func (helper DCBConstitutionHelper) CreatePunishDecryptTx(paymentAddress privacy.PaymentAddress) metadata.Metadata {
	return metadata.NewPunishDCBDecryptMetadata(paymentAddress)
}

func (helper GOVConstitutionHelper) CreatePunishDecryptTx(paymentAddress privacy.PaymentAddress) metadata.Metadata {
	return metadata.NewPunishGOVDecryptMetadata(paymentAddress)
}

func (helper DCBConstitutionHelper) GetSealerPaymentAddress(tx metadata.Transaction) []privacy.PaymentAddress {
	meta := tx.GetMetadata().(*metadata.SealedLv3DCBVoteProposalMetadata)
	return meta.SealedLv3VoteProposalMetadata.SealedVoteProposal.LockerPaymentAddress
}

func (helper GOVConstitutionHelper) GetSealerPaymentAddress(tx metadata.Transaction) []privacy.PaymentAddress {
	meta := tx.GetMetadata().(*metadata.SealedLv3GOVVoteProposalMetadata)
	return meta.SealedLv3VoteProposalMetadata.SealedVoteProposal.LockerPaymentAddress
}

func (helper DCBConstitutionHelper) NewTxRewardProposalSubmitter(blockgen *BlkTmplGenerator, receiverAddress *privacy.PaymentAddress, minerPrivateKey *privacy.SpendingKey) (metadata.Transaction, error) {
	meta := metadata.NewRewardDCBProposalSubmitterMetadata()
	tx := transaction.Tx{}
	err := tx.InitTxSalary(common.RewardProposalSubmitter, receiverAddress, minerPrivateKey, blockgen.chain.config.DataBase, meta)
	if err != nil {
		return nil, err
	}
	return &tx, nil
}

func (helper GOVConstitutionHelper) NewTxRewardProposalSubmitter(blockgen *BlkTmplGenerator, receiverAddress *privacy.PaymentAddress, minerPrivateKey *privacy.SpendingKey) (metadata.Transaction, error) {
	meta := metadata.NewRewardGOVProposalSubmitterMetadata()
	tx := transaction.Tx{}
	err := tx.InitTxSalary(common.RewardProposalSubmitter, receiverAddress, minerPrivateKey, blockgen.chain.config.DataBase, meta)
	if err != nil {
		return nil, err
	}
	return &tx, nil
}

func (helper DCBConstitutionHelper) GetPaymentAddressFromSubmitProposalMetadata(tx metadata.Transaction) *privacy.PaymentAddress {
	meta := tx.GetMetadata().(*metadata.SubmitDCBProposalMetadata)
	return &meta.PaymentAddress
}
func (helper GOVConstitutionHelper) GetPaymentAddressFromSubmitProposalMetadata(tx metadata.Transaction) *privacy.PaymentAddress {
	meta := tx.GetMetadata().(*metadata.SubmitGOVProposalMetadata)
	return &meta.PaymentAddress
}

func (helper DCBConstitutionHelper) GetPaymentAddressVoter(blockgen *BlkTmplGenerator, shardID byte) (privacy.PaymentAddress, error) {
	// bestBlock := blockgen.chain.BestState[shardID].BestBlock
	// _, _, _, tx, _ := blockgen.chain.GetTransactionByHash(&bestBlock.Header.DCBConstitution.AcceptProposalTXID)
	// meta := tx.GetMetadata().(*metadata.AcceptDCBProposalMetadata)
	// return meta.Voter.PaymentAddress, nil
	return privacy.PaymentAddress{}, nil
}
func (helper GOVConstitutionHelper) GetPaymentAddressVoter(blockgen *BlkTmplGenerator, shardID byte) (privacy.PaymentAddress, error) {
	// bestBlock := blockgen.chain.BestState[shardID].BestBlock
	// _, _, _, tx, _ := blockgen.chain.GetTransactionByHash(&bestBlock.Header.GOVConstitution.AcceptProposalTXID)
	// meta := tx.GetMetadata().(*metadata.AcceptGOVProposalMetadata)
	// return meta.Voter.PaymentAddress, nil
	return privacy.PaymentAddress{}, nil
}

func (helper DCBConstitutionHelper) GetPrizeProposal() uint32 {
	return uint32(common.Maxint32(GetOracleDCBNationalWelfare(), int32(0)))
}

func (helper GOVConstitutionHelper) GetPrizeProposal() uint32 {
	return uint32(common.Maxint32(GetOracleGOVNationalWelfare(), int32(0)))
}

func (helper DCBConstitutionHelper) GetTopMostVoteGovernor(blockgen *BlkTmplGenerator) (database.CandidateList, error) {
	return blockgen.chain.config.DataBase.GetTopMostVoteGovernor(helper.GetBoardType(), blockgen.chain.GetCurrentBoardIndex(helper))
}
func (helper GOVConstitutionHelper) GetTopMostVoteGovernor(blockgen *BlkTmplGenerator) (database.CandidateList, error) {
	return blockgen.chain.config.DataBase.GetTopMostVoteGovernor(helper.GetBoardType(), blockgen.chain.GetCurrentBoardIndex(helper))
}

func (helper DCBConstitutionHelper) GetBoardSumToken(blockgen *BlkTmplGenerator) uint64 {
	// return blockgen.chain.BestState[0].BestBlock.Header.DCBGovernor.StartAmountToken
	return 0
}

func (helper GOVConstitutionHelper) GetBoardSumToken(blockgen *BlkTmplGenerator) uint64 {
	// return blockgen.chain.BestState[0].BestBlock.Header.GOVGovernor.StartAmountToken
	return 0
}

func (helper DCBConstitutionHelper) GetBoardFund(blockgen *BlkTmplGenerator) uint64 {
	// return blockgen.chain.BestState[0].BestBlock.Header.BankFund
	return 0
}

func (helper GOVConstitutionHelper) GetBoardFund(blockgen *BlkTmplGenerator) uint64 {
	// return blockgen.chain.BestState[0].BestBlock.Header.SalaryFund
	return 0
}

func (helper DCBConstitutionHelper) GetTokenID() *common.Hash {
	id := common.Hash(common.DCBTokenID)
	return &id
}

func (helper GOVConstitutionHelper) GetTokenID() *common.Hash {
	id := common.Hash(common.GOVTokenID)
	return &id
}

func (helper DCBConstitutionHelper) GetBoard(chain *BlockChain) Governor {
	// return chain.BestState[0].BestBlock.Header.DCBGovernor
	return GovernorInfo{}
}

func (helper GOVConstitutionHelper) GetBoard(chain *BlockChain) Governor {
	// return chain.BestState[0].BestBlock.Header.GOVGovernor
	return GovernorInfo{}
}

func (helper DCBConstitutionHelper) GetAmountVoteTokenOfBoard(blockgen *BlkTmplGenerator, paymentAddress privacy.PaymentAddress, boardIndex uint32) uint64 {
	value, _ := blockgen.chain.config.DataBase.GetVoteTokenAmount(helper.GetBoardType(), boardIndex, paymentAddress)
	return uint64(value)
}
func (helper GOVConstitutionHelper) GetAmountVoteTokenOfBoard(blockgen *BlkTmplGenerator, paymentAddress privacy.PaymentAddress, boardIndex uint32) uint64 {
	value, _ := blockgen.chain.config.DataBase.GetVoteTokenAmount(helper.GetBoardType(), boardIndex, paymentAddress)
	return uint64(value)
}

func (helper DCBConstitutionHelper) GetAmountOfVoteToBoard(blockgen *BlkTmplGenerator, candidatePaymentAddress privacy.PaymentAddress, voterPaymentAddress privacy.PaymentAddress, boardIndex uint32) uint64 {
	key := lvdb.GetKeyVoteBoardList(helper.GetBoardType(), boardIndex, &candidatePaymentAddress, &voterPaymentAddress)
	value, _ := blockgen.chain.config.DataBase.Get(key)
	amount := lvdb.ParseValueVoteBoardList(value)
	return amount
}
func (helper GOVConstitutionHelper) GetAmountOfVoteToBoard(blockgen *BlkTmplGenerator, candidatePaymentAddress privacy.PaymentAddress, voterPaymentAddress privacy.PaymentAddress, boardIndex uint32) uint64 {
	key := lvdb.GetKeyVoteBoardList(helper.GetBoardType(), boardIndex, &candidatePaymentAddress, &voterPaymentAddress)
	value, _ := blockgen.chain.config.DataBase.Get(key)
	amount := lvdb.ParseValueVoteBoardList(value)
	return amount
}

func (helper DCBConstitutionHelper) GetCurrentBoardPaymentAddress(blockgen *BlkTmplGenerator) []privacy.PaymentAddress {
	// return blockgen.chain.BestState[0].BestBlock.Header.DCBGovernor.BoardPaymentAddress
	return []privacy.PaymentAddress{}
}

func (helper GOVConstitutionHelper) GetCurrentBoardPaymentAddress(blockgen *BlkTmplGenerator) []privacy.PaymentAddress {
	// return blockgen.chain.BestState[0].BestBlock.Header.GOVGovernor.BoardPaymentAddress
	return []privacy.PaymentAddress{}
}

func (helper DCBConstitutionHelper) GetConstitutionInfo(chain *BlockChain) ConstitutionInfo {
	// return chain.BestState[0].BestBlock.Header.DCBConstitution.ConstitutionInfo
	return ConstitutionInfo{}
}

func (helper GOVConstitutionHelper) GetConstitutionInfo(chain *BlockChain) ConstitutionInfo {
	// return chain.BestState[0].BestBlock.Header.GOVConstitution.ConstitutionInfo
	return ConstitutionInfo{}
}

func (helper DCBConstitutionHelper) GetCurrentNationalWelfare(chain *BlockChain) int32 {
	return GetOracleDCBNationalWelfare()
}

func (helper GOVConstitutionHelper) GetCurrentNationalWelfare(chain *BlockChain) int32 {
	return GetOracleGOVNationalWelfare()
}

func (helper DCBConstitutionHelper) GetThresholdRatioOfCrisis() int32 {
	return ThresholdRatioOfDCBCrisis
}

func (helper GOVConstitutionHelper) GetThresholdRatioOfCrisis() int32 {
	return ThresholdRatioOfGOVCrisis
}

func (helper DCBConstitutionHelper) GetOldNationalWelfare(chain *BlockChain) int32 {
	// return chain.BestState[0].BestBlock.Header.DCBConstitution.CurrentDCBNationalWelfare
	return 0
}

func (helper GOVConstitutionHelper) GetOldNationalWelfare(chain *BlockChain) int32 {
	// return chain.BestState[0].BestBlock.Header.GOVConstitution.CurrentGOVNationalWelfare
	return 0
}
