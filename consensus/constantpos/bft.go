package constantpos

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/pkg/errors"

	"github.com/ninjadotorg/constant/common"

	"github.com/ninjadotorg/constant/cashec"

	"github.com/ninjadotorg/constant/blockchain"
	"github.com/ninjadotorg/constant/wire"
)

type BFTProtocol struct {
	cBFTMsg   chan wire.Message
	EngineCfg *EngineConfig

	BlockGen   *blockchain.BlkTmplGenerator
	Chain      *blockchain.BlockChain
	Server     serverInterface
	UserKeySet *cashec.KeySet

	cQuit    chan struct{}
	cTimeout chan struct{}

	phase string

	pendingBlock interface{}

	RoleData struct {
		IsProposer bool
		Layer      string
		ShardID    byte
		Committee  []string
	}

	multiSigScheme *multiSigScheme
}

func (protocol *BFTProtocol) Start(isProposer bool, layer string, shardID byte) (interface{}, error) {

	protocol.phase = "listen"
	if isProposer {
		protocol.phase = "propose"
	}
	Logger.log.Info("Starting PBFT protocol for " + layer)
	protocol.multiSigScheme = new(multiSigScheme)
	protocol.multiSigScheme.Init(protocol.UserKeySet, protocol.RoleData.Committee)
	err := protocol.multiSigScheme.Prepare()
	if err != nil {
		return nil, err
	}
	for {
		fmt.Println("New Phase")
		protocol.cTimeout = make(chan struct{})
		select {
		case <-protocol.cQuit:
			return nil, errors.New("Consensus quit")
		default:
			switch protocol.phase {
			case "propose":
				timeout := time.AfterFunc(ListenTimeout*time.Second, func() {
					fmt.Println("Propose phase timeout")
					close(protocol.cTimeout)
				})
				var (
					msg           wire.Message
					readyMsgCount int
				)
				if layer == "beacon" {
					time.Sleep(5 * time.Second) //single-node
					newBlock, err := protocol.BlockGen.NewBlockBeacon(&protocol.UserKeySet.PaymentAddress, &protocol.UserKeySet.PrivateKey)
					if err != nil {
						return nil, err
					}
					jsonBlock, _ := json.Marshal(newBlock)
					msg, err = MakeMsgBFTPropose(jsonBlock)
					if err != nil {
						return nil, err
					}
					protocol.pendingBlock = newBlock
					protocol.multiSigScheme.dataToSig = newBlock.Header.Hash()

					return newBlock, nil //single-node
				} else {
					newBlock, err := protocol.BlockGen.NewBlockShard(&protocol.UserKeySet.PaymentAddress, &protocol.UserKeySet.PrivateKey, shardID)
					if err != nil {
						return nil, err
					}
					jsonBlock, _ := json.Marshal(newBlock)
					msg, err = MakeMsgBFTPropose(jsonBlock)
					if err != nil {
						return nil, err
					}
					protocol.pendingBlock = newBlock
					protocol.multiSigScheme.dataToSig = newBlock.Header.Hash()

					return newBlock, nil //single-node
				}

				fmt.Println()
				fmt.Println("Listen for ready msg")
				fmt.Println()
			proposephase:
				for {
					select {
					case msgReady := <-protocol.cBFTMsg:
						if msgReady.MessageType() == wire.CmdBFTReady {
							var bestStateHash common.Hash
							if layer == "beacon" {
								bestStateHash = protocol.Chain.BestState.Beacon.Hash()
							} else {
								bestStateHash = protocol.Chain.BestState.Shard[shardID].Hash()
							}
							if msgReady.(*wire.MessageBFTReady).BestStateHash == bestStateHash {
								readyMsgCount++
								if readyMsgCount >= (2*len(protocol.RoleData.Committee)/3)-1 {
									timeout.Stop()
									fmt.Println("Collected enough ready")
									select {
									case <-protocol.cTimeout:
										continue
									default:
										close(protocol.cTimeout)
									}
								}
							}
						}
					case <-protocol.cTimeout:
						if readyMsgCount >= (2*len(protocol.RoleData.Committee)/3)-1 {
							<-time.After(2 * time.Second)

							fmt.Println("Propose block")
							if layer == "beacon" {
								go protocol.Server.PushMessageToBeacon(msg)
							} else {
								go protocol.Server.PushMessageToShard(msg, shardID)
							}
							protocol.phase = "prepare"
						} else {
							return nil, errors.New("Didn't received enough ready msg")
						}
						break proposephase
					}
				}
			case "listen":
				fmt.Println("Listen phase")
				timeout := time.AfterFunc(ListenTimeout*time.Second, func() {
					fmt.Println("Listen phase timeout")
					close(protocol.cTimeout)
				})
			listenphase:
				for {
					select {
					case msgPropose := <-protocol.cBFTMsg:
						if msgPropose.MessageType() == wire.CmdBFTPropose {
							fmt.Println("Propose block received")
							if layer == "beacon" {
								pendingBlk := blockchain.BeaconBlock{}
								pendingBlk.UnmarshalJSON(msgPropose.(*wire.MessageBFTPropose).Block)
								blkHash := pendingBlk.Header.Hash()
								err := cashec.ValidateDataB58(pendingBlk.Header.Producer, pendingBlk.ProducerSig, blkHash.GetBytes())
								if err != nil {
									Logger.log.Error(err)
									continue
								}
								err = protocol.Chain.VerifyPreSignBeaconBlock(&pendingBlk)
								if err != nil {
									Logger.log.Error(err)
									continue
								}
								protocol.pendingBlock = &pendingBlk
								protocol.multiSigScheme.dataToSig = pendingBlk.Header.Hash()
							} else {
								pendingBlk := blockchain.ShardBlock{}
								pendingBlk.UnmarshalJSON(msgPropose.(*wire.MessageBFTPropose).Block)
								blkHash := pendingBlk.Header.Hash()
								err := cashec.ValidateDataB58(pendingBlk.Header.Producer, pendingBlk.ProducerSig, blkHash.GetBytes())
								if err != nil {
									Logger.log.Error(err)
									continue
								}
								err = protocol.Chain.VerifyPreSignShardBlock(&pendingBlk, protocol.RoleData.ShardID)
								if err != nil {
									Logger.log.Error(err)
									continue
								}
								protocol.pendingBlock = &pendingBlk
								protocol.multiSigScheme.dataToSig = pendingBlk.Header.Hash()
							}

							protocol.phase = "prepare"
							timeout.Stop()
							break listenphase
						}
					case <-protocol.cTimeout:
						return nil, errors.New("Listen phase timeout")
					}
				}
			case "prepare":
				fmt.Println("Prepare phase")
				time.AfterFunc(PrepareTimeout*time.Second, func() {
					fmt.Println("Prepare phase timeout")
					close(protocol.cTimeout)
				})
				time.AfterFunc(1500*time.Millisecond, func() {
					fmt.Println("Sending out prepare msg")
					msg, err := MakeMsgBFTPrepare(protocol.multiSigScheme.personal.Ri, protocol.UserKeySet.GetPublicKeyB58(), protocol.multiSigScheme.dataToSig.String())
					if err != nil {
						Logger.log.Error(err)
						return
					}
					if layer == "beacon" {
						protocol.Server.PushMessageToBeacon(msg)
					} else {
						protocol.Server.PushMessageToShard(msg, shardID)
					}
				})

				var collectedRiList map[string][]byte //map of members and their Ri
				collectedRiList = make(map[string][]byte)
				collectedRiList[protocol.UserKeySet.GetPublicKeyB58()] = protocol.multiSigScheme.personal.Ri
			preparephase:
				for {
					select {
					case msgPrepare := <-protocol.cBFTMsg:
						if msgPrepare.MessageType() == wire.CmdBFTPrepare {
							fmt.Println("Prepare msg received")
							if common.IndexOfStr(msgPrepare.(*wire.MessageBFTPrepare).Pubkey, protocol.RoleData.Committee) >= 0 && (protocol.multiSigScheme.dataToSig.String() == msgPrepare.(*wire.MessageBFTPrepare).BlkHash) {
								collectedRiList[msgPrepare.(*wire.MessageBFTPrepare).Pubkey] = msgPrepare.(*wire.MessageBFTPrepare).Ri
							}
						}
					case <-protocol.cTimeout:
						//Use collected Ri to calc R & get ValidatorsIdx if len(Ri) > 1/2size(committee)
						// then sig block with this R
						if len(collectedRiList) < (len(protocol.RoleData.Committee) >> 1) {
							return nil, errors.New("Didn't receive enough Ri to continue")
						}
						err := protocol.multiSigScheme.SignData(collectedRiList)
						if err != nil {
							return nil, err
						}

						protocol.phase = "commit"
						break preparephase
					}
				}
			case "commit":
				fmt.Println("Commit phase")
				cmTimeout := time.AfterFunc(CommitTimeout*time.Second, func() {
					fmt.Println("Commit phase timeout")
					close(protocol.cTimeout)
				})

				time.AfterFunc(1500*time.Millisecond, func() {
					msg, err := MakeMsgBFTCommit(protocol.multiSigScheme.combine.CommitSig, protocol.multiSigScheme.combine.R, protocol.multiSigScheme.combine.ValidatorsIdxR, protocol.UserKeySet.GetPublicKeyB58())
					if err != nil {
						Logger.log.Error(err)
						return
					}
					fmt.Println("Sending out commit msg")
					if layer == "beacon" {
						protocol.Server.PushMessageToBeacon(msg)
					} else {
						protocol.Server.PushMessageToShard(msg, shardID)
					}
				})
				var phaseData struct {
					Sigs map[string][]bftCommittedSig
				}

				phaseData.Sigs = make(map[string][]bftCommittedSig)
				phaseData.Sigs[protocol.multiSigScheme.combine.R] = append(phaseData.Sigs[protocol.multiSigScheme.combine.R], bftCommittedSig{
					Pubkey:         protocol.UserKeySet.GetPublicKeyB58(),
					Sig:            protocol.multiSigScheme.combine.CommitSig,
					ValidatorsIdxR: protocol.multiSigScheme.combine.ValidatorsIdxR,
				})
				// commitphase:
				for {
					select {
					case <-protocol.cTimeout:
						//Combine collected Sigs with the same R that has the longest list must has size > 1/2size(committee)
						var szRCombined string
						szRCombined = "1"
						for szR := range phaseData.Sigs {
							if len(phaseData.Sigs[szR]) > (len(protocol.RoleData.Committee) >> 1) {
								if len(szRCombined) == 1 {
									szRCombined = szR
								} else {
									if len(phaseData.Sigs[szR]) > len(phaseData.Sigs[szRCombined]) {
										szRCombined = szR
									}
								}
							}
						}
						if len(szRCombined) == 1 {
							return nil, errors.New("Not enough sigs to combine")
						}

						AggregatedSig, err := protocol.multiSigScheme.CombineSigs(szRCombined, phaseData.Sigs[szRCombined])
						if err != nil {
							return nil, err
						}
						ValidatorsIdxAggSig := make([]int, len(protocol.multiSigScheme.combine.ValidatorsIdxAggSig))
						ValidatorsIdxR := make([]int, len(protocol.multiSigScheme.combine.ValidatorsIdxR))
						copy(ValidatorsIdxAggSig, protocol.multiSigScheme.combine.ValidatorsIdxAggSig)
						copy(ValidatorsIdxR, protocol.multiSigScheme.combine.ValidatorsIdxR)

						fmt.Println("\n \n Block consensus reach", ValidatorsIdxR, ValidatorsIdxAggSig, AggregatedSig)

						if layer == "beacon" {
							protocol.pendingBlock.(*blockchain.BeaconBlock).R = protocol.multiSigScheme.combine.R
							protocol.pendingBlock.(*blockchain.BeaconBlock).AggregatedSig = AggregatedSig
							protocol.pendingBlock.(*blockchain.BeaconBlock).ValidatorsIdx = make([][]int, 2)
							protocol.pendingBlock.(*blockchain.BeaconBlock).ValidatorsIdx[0] = make([]int, len(ValidatorsIdxR))
							protocol.pendingBlock.(*blockchain.BeaconBlock).ValidatorsIdx[1] = make([]int, len(ValidatorsIdxAggSig))
							copy(protocol.pendingBlock.(*blockchain.BeaconBlock).ValidatorsIdx[0], ValidatorsIdxR)
							copy(protocol.pendingBlock.(*blockchain.BeaconBlock).ValidatorsIdx[1], ValidatorsIdxAggSig)
						} else {
							protocol.pendingBlock.(*blockchain.ShardBlock).R = protocol.multiSigScheme.combine.R
							protocol.pendingBlock.(*blockchain.ShardBlock).AggregatedSig = AggregatedSig
							protocol.pendingBlock.(*blockchain.ShardBlock).ValidatorsIdx = make([][]int, 2)
							protocol.pendingBlock.(*blockchain.ShardBlock).ValidatorsIdx[0] = make([]int, len(ValidatorsIdxR))
							protocol.pendingBlock.(*blockchain.ShardBlock).ValidatorsIdx[1] = make([]int, len(ValidatorsIdxAggSig))
							copy(protocol.pendingBlock.(*blockchain.ShardBlock).ValidatorsIdx[0], ValidatorsIdxR)
							copy(protocol.pendingBlock.(*blockchain.ShardBlock).ValidatorsIdx[1], ValidatorsIdxAggSig)
						}

						return protocol.pendingBlock, nil
					case msgCommit := <-protocol.cBFTMsg:
						if msgCommit.MessageType() == wire.CmdBFTCommit {
							fmt.Println("Commit msg received")
							newSig := bftCommittedSig{
								Pubkey:         msgCommit.(*wire.MessageBFTCommit).Pubkey,
								ValidatorsIdxR: msgCommit.(*wire.MessageBFTCommit).ValidatorsIdx,
								Sig:            msgCommit.(*wire.MessageBFTCommit).CommitSig,
							}
							R := msgCommit.(*wire.MessageBFTCommit).R
							err := protocol.multiSigScheme.VerifyCommitSig(newSig.Pubkey, newSig.Sig, R, newSig.ValidatorsIdxR)
							if err != nil {
								return nil, err
							}
							phaseData.Sigs[R] = append(phaseData.Sigs[R], newSig)
							if len(phaseData.Sigs[R]) >= (2 * len(protocol.RoleData.Committee) / 3) {
								cmTimeout.Stop()
								fmt.Println("Collected enough R")
								select {
								case <-protocol.cTimeout:
									continue
								default:
									close(protocol.cTimeout)
								}
							}
						}
					}
				}
			}
		}
	}
}
