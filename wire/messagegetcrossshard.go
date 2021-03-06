package wire

import (
	"encoding/json"

	"github.com/libp2p/go-libp2p-peer"
	"github.com/ninjadotorg/constant/cashec"
	"github.com/ninjadotorg/constant/common"
)

type MessageGetCrossShard struct {
	BlockHash   common.Hash
	FromShardID byte
	ToShardID   byte
	SenderID    string
	Timestamp   int64
}

func (msg *MessageGetCrossShard) Hash() string {
	rawBytes, err := msg.JsonSerialize()
	if err != nil {
		return ""
	}
	return common.HashH(rawBytes).String()
}

func (msg *MessageGetCrossShard) MessageType() string {
	return CmdGetShardToBeacon
}

func (msg *MessageGetCrossShard) MaxPayloadLength(pver int) int {
	return MaxBlockPayload
}

func (msg *MessageGetCrossShard) JsonSerialize() ([]byte, error) {
	jsonBytes, err := json.Marshal(msg)
	return jsonBytes, err
}

func (msg *MessageGetCrossShard) JsonDeserialize(jsonStr string) error {
	err := json.Unmarshal([]byte(jsonStr), msg)
	return err
}

func (msg *MessageGetCrossShard) SetSenderID(senderID peer.ID) error {
	msg.SenderID = senderID.Pretty()
	return nil
}

func (msg *MessageGetCrossShard) SignMsg(_ *cashec.KeySet) error {
	return nil
}

func (msg *MessageGetCrossShard) VerifyMsgSanity() error {
	return nil
}
