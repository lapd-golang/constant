package rpcserver

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/ninjadotorg/constant/metadata"
	"github.com/ninjadotorg/constant/privacy"

	"github.com/ninjadotorg/constant/cashec"
	"github.com/ninjadotorg/constant/common"
	"github.com/ninjadotorg/constant/common/base58"
	"github.com/ninjadotorg/constant/rpcserver/jsonresult"
	"github.com/ninjadotorg/constant/transaction"
	"github.com/ninjadotorg/constant/wallet"
	"github.com/ninjadotorg/constant/wire"
)

/*
// handleList returns a slice of objects representing the wallet
// transactions fitting the given criteria. The confirmations will be more than
// minconf, less than maxconf and if addresses is populated only the addresses
// contained within it will be considered.  If we know nothing about a
// transaction an empty array will be returned.
// params:
Parameter #1—the minimum number of confirmations an output must have
Parameter #2—the maximum number of confirmations an output may have
Parameter #3—the list readonly which be used to view utxo
*/
func (rpcServer RpcServer) handleListOutputCoins(params interface{}, closeChan <-chan struct{}) (interface{}, *RPCError) {
	Logger.log.Info(params)
	result := jsonresult.ListUnspentResult{
		ListUnspentResultItems: make(map[string][]jsonresult.ListUnspentResultItem),
	}

	// get params
	paramsArray := common.InterfaceSlice(params)
	if len(paramsArray) < 1 {
		return nil, NewRPCError(ErrRPCInvalidParams, errors.New("invalid list Key params"))
	}
	listKeyParams := common.InterfaceSlice(paramsArray[0])
	for _, keyParam := range listKeyParams {
		keys := keyParam.(map[string]interface{})

		// get keyset only contain readonly-key by deserializing
		readonlyKeyStr := keys["ReadonlyKey"].(string)
		readonlyKey, err := wallet.Base58CheckDeserialize(readonlyKeyStr)
		if err != nil {
			return nil, NewRPCError(ErrUnexpected, err)
		}

		// get keyset only contain pub-key by deserializing
		pubKeyStr := keys["PaymentAddress"].(string)
		pubKey, err := wallet.Base58CheckDeserialize(pubKeyStr)
		if err != nil {
			return nil, NewRPCError(ErrUnexpected, err)
		}

		// create a key set
		keySet := cashec.KeySet{
			ReadonlyKey:    readonlyKey.KeySet.ReadonlyKey,
			PaymentAddress: pubKey.KeySet.PaymentAddress,
		}
		lastByte := keySet.PaymentAddress.Pk[len(keySet.PaymentAddress.Pk)-1]
		shardIDSender := common.GetShardIDFromLastByte(lastByte)
		constantTokenID := &common.Hash{}
		constantTokenID.SetBytes(common.ConstantID[:])
		outputCoins, err := rpcServer.config.BlockChain.GetListOutputCoinsByKeyset(&keySet, shardIDSender, constantTokenID)
		if err != nil {
			return nil, NewRPCError(ErrUnexpected, err)
		}
		listTxs := make([]jsonresult.ListUnspentResultItem, 0)
		item := jsonresult.ListUnspentResultItem{
			OutCoins: make([]jsonresult.OutCoin, 0),
		}
		for _, outCoin := range outputCoins {
			item.OutCoins = append(item.OutCoins, jsonresult.OutCoin{
				SerialNumber:   base58.Base58Check{}.Encode(outCoin.CoinDetails.SerialNumber.Compress(), common.ZeroByte),
				PublicKey:      base58.Base58Check{}.Encode(outCoin.CoinDetails.PublicKey.Compress(), common.ZeroByte),
				Value:          outCoin.CoinDetails.Value,
				Info:           base58.Base58Check{}.Encode(outCoin.CoinDetails.Info[:], common.ZeroByte),
				CoinCommitment: base58.Base58Check{}.Encode(outCoin.CoinDetails.CoinCommitment.Compress(), common.ZeroByte),
				Randomness:     *outCoin.CoinDetails.Randomness,
				SNDerivator:    *outCoin.CoinDetails.SNDerivator,
			})
			listTxs = append(listTxs, item)
			result.ListUnspentResultItems[readonlyKeyStr] = listTxs
		}
	}

	return result, nil
}

/*
// handleCreateTransaction handles createtransaction commands.
*/
func (rpcServer RpcServer) handleCreateRawTransaction(params interface{}, closeChan <-chan struct{}) (interface{}, *RPCError) {
	var err error
	tx, err := rpcServer.buildRawTransaction(params, nil)
	if err.(*RPCError) != nil {
		Logger.log.Critical(err)
		return nil, NewRPCError(ErrCreateTxData, err)
	}
	byteArrays, err := json.Marshal(tx)
	if err != nil {
		// return hex for a new tx
		return nil, NewRPCError(ErrCreateTxData, err)
	}
	txShardID := common.GetShardIDFromLastByte(tx.GetSenderAddrLastByte())
	result := jsonresult.CreateTransactionResult{
		TxID:            tx.Hash().String(),
		Base58CheckData: base58.Base58Check{}.Encode(byteArrays, 0x00),
		ShardID:         txShardID,
	}
	return result, nil
}

/*
// handleSendTransaction implements the sendtransaction command.
Parameter #1—a serialized transaction to broadcast
Parameter #2–whether to allow high fees
Result—a TXID or error Message
*/
func (rpcServer RpcServer) handleSendRawTransaction(params interface{}, closeChan <-chan struct{}) (interface{}, *RPCError) {
	Logger.log.Info(params)
	arrayParams := common.InterfaceSlice(params)
	base58CheckData := arrayParams[0].(string)
	rawTxBytes, _, err := base58.Base58Check{}.Decode(base58CheckData)

	if err != nil {
		return nil, NewRPCError(ErrSendTxData, err)
	}
	var tx transaction.Tx
	// Logger.log.Info(string(rawTxBytes))
	err = json.Unmarshal(rawTxBytes, &tx)
	if err != nil {
		return nil, NewRPCError(ErrSendTxData, err)
	}

	hash, txDesc, err := rpcServer.config.TxMemPool.MaybeAcceptTransaction(&tx)
	if err != nil {
		return nil, NewRPCError(ErrSendTxData, err)
	}

	Logger.log.Infof("there is hash of transaction: %s\n", hash.String())
	Logger.log.Infof("there is priority of transaction in pool: %d", txDesc.StartingPriority)

	// broadcast Message
	txMsg, err := wire.MakeEmptyMessage(wire.CmdTx)
	if err != nil {
		return nil, NewRPCError(ErrSendTxData, err)
	}

	txMsg.(*wire.MessageTx).Transaction = &tx
	err = rpcServer.config.Server.PushMessageToAll(txMsg)
	if err != nil {
		return nil, NewRPCError(ErrSendTxData, err)
	}

	txID := tx.Hash().String()
	result := jsonresult.CreateTransactionResult{
		TxID: txID,
	}
	return result, nil
}

/*
handleCreateAndSendTx - RPC creates transaction and send to network
*/
func (rpcServer RpcServer) handleCreateAndSendTx(params interface{}, closeChan <-chan struct{}) (interface{}, *RPCError) {
	var err error
	data, err := rpcServer.handleCreateRawTransaction(params, closeChan)
	if err.(*RPCError) != nil {
		return nil, NewRPCError(ErrCreateTxData, err)
	}
	tx := data.(jsonresult.CreateTransactionResult)
	base58CheckData := tx.Base58CheckData
	newParam := make([]interface{}, 0)
	newParam = append(newParam, base58CheckData)
	sendResult, err := rpcServer.handleSendRawTransaction(newParam, closeChan)
	if err.(*RPCError) != nil {
		return nil, NewRPCError(ErrSendTxData, err)
	}
	result := jsonresult.CreateTransactionResult{
		TxID:    sendResult.(jsonresult.CreateTransactionResult).TxID,
		ShardID: tx.ShardID,
	}
	return result, nil
}

/*
handleGetMempoolInfo - RPC returns information about the node's current txs memory pool
*/
func (rpcServer RpcServer) handleGetMempoolInfo(params interface{}, closeChan <-chan struct{}) (interface{}, *RPCError) {
	result := jsonresult.GetMempoolInfo{}
	result.Size = rpcServer.config.TxMemPool.Count()
	result.Bytes = rpcServer.config.TxMemPool.Size()
	result.MempoolMaxFee = rpcServer.config.TxMemPool.MaxFee()
	result.ListTxs = rpcServer.config.TxMemPool.ListTxs()
	return result, nil
}

// Get transaction by Hash
func (rpcServer RpcServer) handleGetTransactionByHash(params interface{}, closeChan <-chan struct{}) (interface{}, *RPCError) {
	arrayParams := common.InterfaceSlice(params)
	// param #1: transaction Hash
	if len(arrayParams) < 1 {
		return nil, NewRPCError(ErrRPCInvalidParams, errors.New("Tx hash is empty"))
	}
	Logger.log.Infof("Get TransactionByHash input Param %+v", arrayParams[0].(string))
	txHash, _ := common.Hash{}.NewHashFromStr(arrayParams[0].(string))
	Logger.log.Infof("Get Transaction By Hash %+v", txHash)
	shardID, blockHash, index, tx, err := rpcServer.config.BlockChain.GetTransactionByHash(txHash)
	if err != nil {
		return nil, NewRPCError(ErrUnexpected, err)
	}
	result := jsonresult.TransactionDetail{}
	switch tx.GetType() {
	case common.TxNormalType, common.TxSalaryType:
		{
			tempTx := tx.(*transaction.Tx)
			result = jsonresult.TransactionDetail{
				BlockHash: blockHash.String(),
				Index:     uint64(index),
				ShardID:   shardID,
				Hash:      tx.Hash().String(),
				Version:   tempTx.Version,
				Type:      tempTx.Type,
				LockTime:  time.Unix(tempTx.LockTime, 0).Format(common.DateOutputFormat),
				Fee:       tempTx.Fee,
				Proof:     tempTx.Proof,
				SigPubKey: tempTx.SigPubKey,
				Sig:       tempTx.Sig,
			}
			metaData, _ := json.MarshalIndent(tempTx.Metadata, "", "\t")
			result.Metadata = string(metaData)
		}
	case common.TxCustomTokenType:
		{
			tempTx := tx.(*transaction.TxCustomToken)
			result = jsonresult.TransactionDetail{
				BlockHash: blockHash.String(),
				Index:     uint64(index),
				ShardID:   shardID,
				Hash:      tx.Hash().String(),
				Version:   tempTx.Version,
				Type:      tempTx.Type,
				LockTime:  time.Unix(tempTx.LockTime, 0).Format(common.DateOutputFormat),
				Fee:       tempTx.Fee,
				Proof:     tempTx.Proof,
				SigPubKey: tempTx.SigPubKey,
				Sig:       tempTx.Sig,
			}
			txCustomData, _ := json.MarshalIndent(tempTx.TxTokenData, "", "\t")
			result.CustomTokenData = string(txCustomData)
			if tempTx.Metadata != nil {
				metaData, _ := json.MarshalIndent(tempTx.Metadata, "", "\t")
				result.Metadata = string(metaData)
			}
		}
	case common.TxCustomTokenPrivacyType:
		{
			tempTx := tx.(*transaction.TxCustomTokenPrivacy)
			result = jsonresult.TransactionDetail{
				BlockHash: blockHash.String(),
				Index:     uint64(index),
				ShardID:   shardID,
				Hash:      tx.Hash().String(),
				Version:   tempTx.Version,
				Type:      tempTx.Type,
				LockTime:  time.Unix(tempTx.LockTime, 0).Format(common.DateOutputFormat),
				Fee:       tempTx.Fee,
				Proof:     tempTx.Proof,
				SigPubKey: tempTx.SigPubKey,
				Sig:       tempTx.Sig,
			}
			tokenData, _ := json.MarshalIndent(tempTx.TxTokenPrivacyData, "", "\t")
			result.PrivacyCustomTokenData = string(tokenData)
			if tempTx.Metadata != nil {
				metaData, _ := json.MarshalIndent(tempTx.Metadata, "", "\t")
				result.Metadata = string(metaData)
			}
		}
	default:
		{
			return nil, NewRPCError(ErrTxTypeInvalid, errors.New("Tx type is invalid"))
		}
	}
	return result, nil
}

func (rpcServer RpcServer) handleGetCommitteeCandidateList(params interface{}, closeChan <-chan struct{}) (interface{}, *RPCError) {
	// param #1: private key of sender
	// cndList := self.config.BlockChain.GetCommitteeCandidateList()
	// return cndList, nil
	return nil, nil
}

func (self RpcServer) handleRetrieveCommiteeCandidate(params interface{}, closeChan <-chan struct{}) (interface{}, *RPCError) {
	// candidateInfo := self.config.BlockChain.GetCommitteCandidate(params.(string))
	// if candidateInfo == nil {
	// 	return nil, nil
	// }
	// result := jsonresult.RetrieveCommitteecCandidateResult{}
	// result.Init(candidateInfo)
	// return result, nil
	return nil, nil
}

func (self RpcServer) handleGetBlockProducerList(params interface{}, closeChan <-chan struct{}) (interface{}, *RPCError) {
	result := make(map[string]string)
	// for shardID, bestState := range self.config.BlockChain.BestState {
	// 	if bestState.BestBlock.BlockProducer != "" {
	// 		result[strconv.Itoa(shardID)] = bestState.BestBlock.BlockProducer
	// 	} else {
	// 		result[strconv.Itoa(shardID)] = self.config.ChainParams.GenesisBlock.Header.Committee[shardID]
	// 	}
	// }
	return result, nil
}

// handleCreateRawCustomTokenTransaction - handle create a custom token command and return in hex string format.
func (rpcServer RpcServer) handleCreateRawCustomTokenTransaction(
	params interface{},
	closeChan <-chan struct{},
) (interface{}, *RPCError) {
	var err error
	tx, err := rpcServer.buildRawCustomTokenTransaction(params, nil)
	if err != nil {
		Logger.log.Error(err)
		return nil, NewRPCError(ErrCreateTxData, err)
	}

	byteArrays, err := json.Marshal(tx)
	if err != nil {
		Logger.log.Error(err)
		return nil, NewRPCError(ErrCreateTxData, err)
	}
	result := jsonresult.CreateTransactionResult{
		TxID:            tx.Hash().String(),
		Base58CheckData: base58.Base58Check{}.Encode(byteArrays, 0x00),
	}
	return result, nil
}

// handleSendRawTransaction...
func (rpcServer RpcServer) handleSendRawCustomTokenTransaction(params interface{}, closeChan <-chan struct{}) (interface{}, *RPCError) {
	Logger.log.Info(params)
	arrayParams := common.InterfaceSlice(params)
	base58CheckData := arrayParams[0].(string)
	rawTxBytes, _, err := base58.Base58Check{}.Decode(base58CheckData)

	if err != nil {
		return nil, NewRPCError(ErrSendTxData, err)
	}
	tx := transaction.TxCustomToken{}
	//tx := transaction.TxCustomToken{}
	// Logger.log.Info(string(rawTxBytes))
	err = json.Unmarshal(rawTxBytes, &tx)
	if err != nil {
		return nil, NewRPCError(ErrSendTxData, err)
	}

	hash, txDesc, err := rpcServer.config.TxMemPool.MaybeAcceptTransaction(&tx)
	if err != nil {
		return nil, NewRPCError(ErrSendTxData, err)
	}

	Logger.log.Infof("there is hash of transaction: %s\n", hash.String())
	Logger.log.Infof("there is priority of transaction in pool: %d", txDesc.StartingPriority)

	// broadcast message
	txMsg, err := wire.MakeEmptyMessage(wire.CmdCustomToken)
	if err != nil {
		return nil, NewRPCError(ErrSendTxData, err)
	}

	txMsg.(*wire.MessageTx).Transaction = &tx
	rpcServer.config.Server.PushMessageToAll(txMsg)

	return tx.Hash(), nil
}

// handleCreateAndSendCustomTokenTransaction - create and send a tx which process on a custom token look like erc-20 on eth
func (rpcServer RpcServer) handleCreateAndSendCustomTokenTransaction(params interface{}, closeChan <-chan struct{}) (interface{}, *RPCError) {
	data, err := rpcServer.handleCreateRawCustomTokenTransaction(params, closeChan)
	if err != nil {
		return nil, err
	}
	tx := data.(jsonresult.CreateTransactionResult)
	base58CheckData := tx.Base58CheckData
	if err != nil {
		return nil, err
	}
	newParam := make([]interface{}, 0)
	newParam = append(newParam, base58CheckData)
	txId, err := rpcServer.handleSendRawCustomTokenTransaction(newParam, closeChan)
	return txId, err
}

func (rpcServer RpcServer) handleGetListCustomTokenBalance(params interface{}, closeChan <-chan struct{}) (interface{}, *RPCError) {
	arrayParams := common.InterfaceSlice(params)
	accountParam := arrayParams[0].(string)
	account, err := wallet.Base58CheckDeserialize(accountParam)
	if err != nil {
		return nil, nil
	}
	result := jsonresult.ListCustomTokenBalance{ListCustomTokenBalance: []jsonresult.CustomTokenBalance{}}
	result.PaymentAddress = accountParam
	accountPaymentAddress := account.KeySet.PaymentAddress
	temps, err := rpcServer.config.BlockChain.ListCustomToken()
	if err != nil {
		return nil, NewRPCError(ErrUnexpected, err)
	}
	for _, tx := range temps {
		item := jsonresult.CustomTokenBalance{}
		item.Name = tx.TxTokenData.PropertyName
		item.Symbol = tx.TxTokenData.PropertySymbol
		item.TokenID = tx.TxTokenData.PropertyID.String()
		item.TokenImage = common.Render([]byte(item.TokenID))
		tokenID := tx.TxTokenData.PropertyID
		res, err := rpcServer.config.BlockChain.GetListTokenHolders(&tokenID)
		if err != nil {
			return nil, NewRPCError(ErrUnexpected, err)
		}
		pubkey := base58.Base58Check{}.Encode(accountPaymentAddress.Pk, 0x00)
		item.Amount = res[pubkey]
		if item.Amount == 0 {
			continue
		}
		result.ListCustomTokenBalance = append(result.ListCustomTokenBalance, item)
		result.PaymentAddress = account.Base58CheckSerialize(wallet.PaymentAddressType)
	}
	return result, nil
}

func (rpcServer RpcServer) handleGetListPrivacyCustomTokenBalance(params interface{}, closeChan <-chan struct{}) (interface{}, *RPCError) {
	arrayParams := common.InterfaceSlice(params)
	privateKey := arrayParams[0].(string)
	account, err := wallet.Base58CheckDeserialize(privateKey)
	account.KeySet.ImportFromPrivateKey(&account.KeySet.PrivateKey)
	if err != nil {
		return nil, nil
	}
	result := jsonresult.ListCustomTokenBalance{ListCustomTokenBalance: []jsonresult.CustomTokenBalance{}}
	result.PaymentAddress = account.Base58CheckSerialize(wallet.PaymentAddressType)
	temps, err := rpcServer.config.BlockChain.ListPrivacyCustomToken()
	if err != nil {
		return nil, NewRPCError(ErrUnexpected, err)
	}
	for _, tx := range temps {
		item := jsonresult.CustomTokenBalance{}
		item.Name = tx.TxTokenPrivacyData.PropertyName
		item.Symbol = tx.TxTokenPrivacyData.PropertySymbol
		item.TokenID = tx.TxTokenPrivacyData.PropertyID.String()
		item.TokenImage = common.Render([]byte(item.TokenID))
		tokenID := tx.TxTokenPrivacyData.PropertyID

		balance := uint64(0)
		// get balance for accountName in wallet
		lastByte := account.KeySet.PaymentAddress.Pk[len(account.KeySet.PaymentAddress.Pk)-1]
		shardIDSender := common.GetShardIDFromLastByte(lastByte)
		constantTokenID := &common.Hash{}
		constantTokenID.SetBytes(common.ConstantID[:])
		outcoints, err := rpcServer.config.BlockChain.GetListOutputCoinsByKeyset(&account.KeySet, shardIDSender, &tokenID)
		if err != nil {
			return nil, NewRPCError(ErrUnexpected, err)
		}
		for _, out := range outcoints {
			balance += out.CoinDetails.Value
		}

		item.Amount = balance
		if item.Amount == 0 {
			continue
		}
		item.IsPrivacy = true
		result.ListCustomTokenBalance = append(result.ListCustomTokenBalance, item)
		result.PaymentAddress = account.Base58CheckSerialize(wallet.PaymentAddressType)
	}
	return result, nil
}

// handleCustomTokenDetail - return list tx which relate to custom token by token id
func (rpcServer RpcServer) handleCustomTokenDetail(params interface{}, closeChan <-chan struct{}) (interface{}, *RPCError) {
	arrayParams := common.InterfaceSlice(params)
	tokenID, err := common.Hash{}.NewHashFromStr(arrayParams[0].(string))
	if err != nil {
		return nil, NewRPCError(ErrUnexpected, err)
	}
	txs, _ := rpcServer.config.BlockChain.GetCustomTokenTxsHash(tokenID)
	result := jsonresult.CustomToken{
		ListTxs: []string{},
	}
	for _, tx := range txs {
		result.ListTxs = append(result.ListTxs, tx.String())
	}
	return result, nil
}

// handlePrivacyCustomTokenDetail - return list tx which relate to privacy custom token by token id
func (rpcServer RpcServer) handlePrivacyCustomTokenDetail(params interface{}, closeChan <-chan struct{}) (interface{}, *RPCError) {
	arrayParams := common.InterfaceSlice(params)
	tokenID, err := common.Hash{}.NewHashFromStr(arrayParams[0].(string))
	if err != nil {
		return nil, NewRPCError(ErrUnexpected, err)
	}
	txs, _ := rpcServer.config.BlockChain.GetPrivacyCustomTokenTxsHash(tokenID)
	result := jsonresult.CustomToken{
		ListTxs: []string{},
	}
	for _, tx := range txs {
		result.ListTxs = append(result.ListTxs, tx.String())
	}
	return result, nil
}

func (rpcServer RpcServer) handleListUnspentCustomTokenTransaction(params interface{}, closeChan <-chan struct{}) (interface{}, *RPCError) {
	arrayParams := common.InterfaceSlice(params)
	// param #1: paymentaddress of sender
	senderKeyParam := arrayParams[0]
	senderKey, err := wallet.Base58CheckDeserialize(senderKeyParam.(string))
	if err != nil {
		return nil, NewRPCError(ErrUnexpected, err)
	}
	senderKeyset := senderKey.KeySet

	// param #2: tokenID
	tokenIDParam := arrayParams[1]
	tokenID, _ := common.Hash{}.NewHashFromStr(tokenIDParam.(string))
	unspentTxTokenOuts, err := rpcServer.config.BlockChain.GetUnspentTxCustomTokenVout(senderKeyset, tokenID)
	if err != nil {
		return nil, NewRPCError(ErrUnexpected, err)
	}
	return unspentTxTokenOuts, NewRPCError(ErrUnexpected, err)
}

// handleCreateSignatureOnCustomTokenTx - return a signature which is signed on raw custom token tx
func (rpcServer RpcServer) handleCreateSignatureOnCustomTokenTx(params interface{}, closeChan <-chan struct{}) (interface{}, *RPCError) {
	Logger.log.Info(params)
	arrayParams := common.InterfaceSlice(params)
	base58CheckDate := arrayParams[0].(string)
	rawTxBytes, _, err := base58.Base58Check{}.Decode(base58CheckDate)

	if err != nil {
		return nil, NewRPCError(ErrCreateTxData, err)
	}
	tx := transaction.TxCustomToken{}
	// Logger.log.Info(string(rawTxBytes))
	err = json.Unmarshal(rawTxBytes, &tx)
	if err != nil {
		return nil, NewRPCError(ErrCreateTxData, err)
	}
	senderKeyParam := arrayParams[1]
	senderKey, err := wallet.Base58CheckDeserialize(senderKeyParam.(string))
	if err != nil {
		return nil, NewRPCError(ErrCreateTxData, err)
	}
	senderKey.KeySet.ImportFromPrivateKey(&senderKey.KeySet.PrivateKey)
	if err != nil {
		return nil, NewRPCError(ErrCreateTxData, err)
	}

	jsSignByteArray, err := tx.GetTxCustomTokenSignature(senderKey.KeySet)
	if err != nil {
		return nil, NewRPCError(ErrCreateTxData, errors.New("failed to sign the custom token"))
	}
	return hex.EncodeToString(jsSignByteArray), nil
}

// handleRandomCommitments - from input of outputcoin, random to create data for create new tx
func (rpcServer RpcServer) handleRandomCommitments(params interface{}, closeChan <-chan struct{}) (interface{}, *RPCError) {
	arrayParams := common.InterfaceSlice(params)

	// #1: payment address
	paymentAddressStr := arrayParams[0].(string)
	key, err := wallet.Base58CheckDeserialize(paymentAddressStr)
	if err != nil {
		return nil, NewRPCError(ErrUnexpected, err)
	}
	lastByte := key.KeySet.PaymentAddress.Pk[len(key.KeySet.PaymentAddress.Pk)-1]
	shardIDSender := common.GetShardIDFromLastByte(lastByte)

	// #2: available inputCoin from old outputcoin
	data := jsonresult.ListUnspentResultItem{}
	data.Init(arrayParams[0])
	usableOutputCoins := []*privacy.OutputCoin{}
	for _, item := range data.OutCoins {
		i := &privacy.OutputCoin{
			CoinDetails: &privacy.Coin{
				Value:       item.Value,
				Randomness:  &item.Randomness,
				SNDerivator: &item.SNDerivator,
			},
		}
		i.CoinDetails.Info, _, _ = base58.Base58Check{}.Decode(item.Info)

		CoinCommitmentBytes, _, _ := base58.Base58Check{}.Decode(item.CoinCommitment)
		CoinCommitment := &privacy.EllipticPoint{}
		_ = CoinCommitment.Decompress(CoinCommitmentBytes)
		i.CoinDetails.CoinCommitment = CoinCommitment

		PublicKeyBytes, _, _ := base58.Base58Check{}.Decode(item.PublicKey)
		PublicKey := &privacy.EllipticPoint{}
		_ = PublicKey.Decompress(PublicKeyBytes)
		i.CoinDetails.PublicKey = PublicKey

		InfoBytes, _, _ := base58.Base58Check{}.Decode(item.Info)
		i.CoinDetails.Info = InfoBytes

		usableOutputCoins = append(usableOutputCoins, i)
	}
	usableInputCoins := transaction.ConvertOutputCoinToInputCoin(usableOutputCoins)
	constantTokenID := &common.Hash{}
	constantTokenID.SetBytes(common.ConstantID[:])
	commitmentIndexs, myCommitmentIndexs := rpcServer.config.BlockChain.RandomCommitmentsProcess(usableInputCoins, 0, shardIDSender, constantTokenID)
	result := make(map[string]interface{})
	result["CommitmentIndices"] = commitmentIndexs
	result["MyCommitmentIndexs"] = myCommitmentIndexs

	return result, nil
}

// handleHasSerialNumbers - check list serial numbers existed in db of node
func (rpcServer RpcServer) handleHasSerialNumbers(params interface{}, closeChan <-chan struct{}) (interface{}, *RPCError) {
	// arrayParams := common.InterfaceSlice(params)

	// // #1: payment address
	// paymentAddressStr := arrayParams[0].(string)
	// key, err := wallet.Base58CheckDeserialize(paymentAddressStr)
	// if err != nil {
	// 	return nil, NewRPCError(ErrUnexpected, err)
	// }
	// lastByte := key.KeySet.PaymentAddress.Pk[len(key.KeySet.PaymentAddress.Pk)-1]
	// shardIDSender, err := common.GetTxSenderShard(lastByte)
	// if err != nil {
	// 	return nil, NewRPCError(ErrUnexpected, err)
	// }
	// //#2: list serialnumbers in base58check encode string
	// serialNumbersStr := arrayParams[1].([]interface{})

	// result := make(map[byte][]string)
	// result[0] = []string{}
	// result[1] = []string{}
	// constantTokenID := &common.Hash{}
	// constantTokenID.SetBytes(common.ConstantID[:])
	// for _, item := range serialNumbersStr {
	// 	serialNumber, _, _ := base58.Base58Check{}.Decode(item.(string))
	// 	db := *(rpcServer.config.Database)
	// 	ok, err := db.HasSerialNumber(constantTokenID, serialNumber, shardIDSender)
	// 	if ok && err != nil {
	// 		result[0] = append(result[0], item.(string))
	// 	} else {
	// 		result[1] = append(result[1], item.(string))
	// 	}
	// }

	// return result, nil
	return nil, nil
}

// handleCreateRawCustomTokenTransaction - handle create a custom token command and return in hex string format.
func (rpcServer RpcServer) handleCreateRawPrivacyCustomTokenTransaction(
	params interface{},
	closeChan <-chan struct{},
) (interface{}, *RPCError) {
	var err error
	tx, err := rpcServer.buildRawPrivacyCustomTokenTransaction(params)
	if err.(*transaction.TransactionError) != nil {
		Logger.log.Error(err)
		return nil, NewRPCError(ErrCreateTxData, err)
	}

	byteArrays, err := json.Marshal(tx)
	if err != nil {
		Logger.log.Error(err)
		return nil, NewRPCError(ErrCreateTxData, err)
	}
	result := jsonresult.CreateTransactionResult{
		TxID:            tx.Hash().String(),
		Base58CheckData: base58.Base58Check{}.Encode(byteArrays, 0x00),
	}
	return result, nil
}

// handleSendRawTransaction...
func (rpcServer RpcServer) handleSendRawPrivacyCustomTokenTransaction(params interface{}, closeChan <-chan struct{}) (interface{}, *RPCError) {
	Logger.log.Info(params)
	arrayParams := common.InterfaceSlice(params)
	base58CheckData := arrayParams[0].(string)
	rawTxBytes, _, err := base58.Base58Check{}.Decode(base58CheckData)

	if err != nil {
		return nil, NewRPCError(ErrSendTxData, err)
	}
	tx := transaction.TxCustomTokenPrivacy{}
	err = json.Unmarshal(rawTxBytes, &tx)
	if err != nil {
		return nil, NewRPCError(ErrSendTxData, err)
	}

	hash, txDesc, err := rpcServer.config.TxMemPool.MaybeAcceptTransaction(&tx)
	if err != nil {
		return nil, NewRPCError(ErrSendTxData, err)
	}

	Logger.log.Infof("there is hash of transaction: %s\n", hash.String())
	Logger.log.Infof("there is priority of transaction in pool: %d", txDesc.StartingPriority)

	// broadcast message
	txMsg, err := wire.MakeEmptyMessage(wire.CmdPrivacyCustomToken)
	if err != nil {
		return nil, NewRPCError(ErrSendTxData, err)
	}

	txMsg.(*wire.MessageTx).Transaction = &tx
	rpcServer.config.Server.PushMessageToAll(txMsg)

	return tx.Hash(), nil
}

// handleCreateAndSendCustomTokenTransaction - create and send a tx which process on a custom token look like erc-20 on eth
func (rpcServer RpcServer) handleCreateAndSendPrivacyCustomTokenTransaction(params interface{}, closeChan <-chan struct{}) (interface{}, *RPCError) {
	data, err := rpcServer.handleCreateRawPrivacyCustomTokenTransaction(params, closeChan)
	if err != nil {
		return nil, err
	}
	tx := data.(jsonresult.CreateTransactionResult)
	base58CheckData := tx.Base58CheckData
	if err != nil {
		return nil, err
	}
	newParam := make([]interface{}, 0)
	newParam = append(newParam, base58CheckData)
	txId, err := rpcServer.handleSendRawPrivacyCustomTokenTransaction(newParam, closeChan)
	return txId, err
}

/*
// handleCreateRawStakingTransaction handles create staking
*/
func (rpcServer RpcServer) handleCreateRawStakingTransaction(params interface{}, closeChan <-chan struct{}) (interface{}, *RPCError) {
	// get params
	paramsArray := common.InterfaceSlice(params)
	if len(paramsArray) < 5 {
		return nil, NewRPCError(ErrRPCInvalidParams, errors.New("Empty staking type params"))
	}
	stakingType, ok := paramsArray[4].(float64)
	if !ok {
		return nil, NewRPCError(ErrRPCInvalidParams, errors.New("Invalid staking type params"))
	}

	var err error
	metadata, err := metadata.NewStakingMetadata(int(stakingType))
	tx, err := rpcServer.buildRawTransaction(params, metadata)
	if err.(*RPCError) != nil {
		Logger.log.Critical(err)
		return nil, NewRPCError(ErrCreateTxData, err)
	}
	byteArrays, err := json.Marshal(tx)
	if err != nil {
		// return hex for a new tx
		return nil, NewRPCError(ErrCreateTxData, err)
	}
	txShardID := common.GetShardIDFromLastByte(tx.GetSenderAddrLastByte())
	result := jsonresult.CreateTransactionResult{
		TxID:            tx.Hash().String(),
		Base58CheckData: base58.Base58Check{}.Encode(byteArrays, 0x00),
		ShardID:         txShardID,
	}
	return result, nil
}

/*
handleCreateAndSendStakingTx - RPC creates staking transaction and send to network
*/
func (rpcServer RpcServer) handleCreateAndSendStakingTx(params interface{}, closeChan <-chan struct{}) (interface{}, *RPCError) {
	var err error
	data, err := rpcServer.handleCreateRawStakingTransaction(params, closeChan)
	if err.(*RPCError) != nil {
		return nil, NewRPCError(ErrCreateTxData, err)
	}
	tx := data.(jsonresult.CreateTransactionResult)
	base58CheckData := tx.Base58CheckData
	newParam := make([]interface{}, 0)
	newParam = append(newParam, base58CheckData)
	sendResult, err := rpcServer.handleSendRawTransaction(newParam, closeChan)
	if err.(*RPCError) != nil {
		return nil, NewRPCError(ErrSendTxData, err)
	}
	result := jsonresult.CreateTransactionResult{
		TxID:    sendResult.(jsonresult.CreateTransactionResult).TxID,
		ShardID: tx.ShardID,
	}
	return result, nil
}
