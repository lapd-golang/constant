package privacy

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
)

func TestEncryptionBytes (t*testing.T){
	msg := RandBytes(100)
	spendingKey := GenerateSpendingKey([]byte{123})
	publicKeyBytes := GeneratePublicKey(spendingKey)

	privateKey := new(big.Int).SetBytes(spendingKey)
	publicKey := new(EllipticPoint)
	err := publicKey.Decompress(publicKeyBytes)
	if err != nil{
		fmt.Printf("ERR: %v\n", err)
	}

	ciphertext, err := AdvanceEncrypt(msg, publicKey)
	if err != nil{
		fmt.Printf("ERR: %v\n", err)
	}

	ciphertextBytes := ciphertext.Bytes()
	ciphertext2 := new(Ciphertext)
	ciphertext2.SetBytes(ciphertextBytes)

	msg2, err := AdvanceDecrypt(ciphertext2, privateKey)
	if err != nil{
		fmt.Printf("ERR: %v\n", err)
	}

	fmt.Printf("msg: %v\n", msg)
	fmt.Printf("msg decypted: %v\n", msg2)

	assert.Equal(t, msg, msg2)
}
