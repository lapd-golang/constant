package wire

import (
	"encoding/json"

	"github.com/libp2p/go-libp2p-peer"
	"github.com/ninjadotorg/constant/cashec"
	"github.com/ninjadotorg/constant/common"
)

const (
	MaxBFTPreparePayload = 1000 // 1 Kb
)

type MessageBFTPrepare struct {
	BlkHash string
	Ri      []byte
	Pubkey  string
	MsgSig  string
}

func (self *MessageBFTPrepare) Hash() string {
	rawBytes, err := self.JsonSerialize()
	if err != nil {
		return ""
	}
	return common.HashH(rawBytes).String()
}

func (self *MessageBFTPrepare) MessageType() string {
	return CmdBFTPrepare
}

func (self *MessageBFTPrepare) MaxPayloadLength(pver int) int {
	return MaxBFTPreparePayload
}

func (self *MessageBFTPrepare) JsonSerialize() ([]byte, error) {
	jsonBytes, err := json.Marshal(self)
	return jsonBytes, err
}

func (self *MessageBFTPrepare) JsonDeserialize(jsonStr string) error {
	err := json.Unmarshal([]byte(jsonStr), self)
	return err
}

func (self *MessageBFTPrepare) SetSenderID(senderID peer.ID) error {
	return nil
}

func (self *MessageBFTPrepare) SignMsg(_ *cashec.KeySet) error {
	return nil
}

func (self *MessageBFTPrepare) VerifyMsgSanity() error {
	return nil
}