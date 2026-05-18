package polymarket

import (
	"crypto/ecdsa"
	"fmt"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
)

// Signer 签名器
type Signer struct {
	privateKey      *ecdsa.PrivateKey
	chainID         int
	address         string // EOA 地址，由私钥推导
	addressOverride string // 地址覆盖（用于 signatureType=3 POLY_1271）
}

// NewSigner 创建新的签名器
func NewSigner(privateKeyHex string, chainID int) (*Signer, error) {
	if privateKeyHex == "" || chainID == 0 {
		return nil, fmt.Errorf("private key and chain ID are required")
	}

	// 移除0x前缀
	if len(privateKeyHex) > 2 && privateKeyHex[:2] == "0x" {
		privateKeyHex = privateKeyHex[2:]
	}

	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %w", err)
	}

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("failed to get public key")
	}

	address := crypto.PubkeyToAddress(*publicKeyECDSA).Hex()

	return &Signer{
		privateKey: privateKey,
		chainID:    chainID,
		address:    address,
	}, nil
}

// Address 返回签名器的地址
// 如果设置了 addressOverride（如 type=3 POLY_1271），返回覆盖地址，
// 否则返回 EOA 地址。
// 对于 type=3，POLY_ADDRESS header 必须用 deposit wallet 地址，
// 因为 API Key 的 owner 是 deposit wallet 地址，而非 EOA 地址。
func (s *Signer) Address() string {
	if s.addressOverride != "" {
		return s.addressOverride
	}
	return s.address
}

// SetAddressOverride 设置地址覆盖
// 用于 signatureType=3 (POLY_1271) 场景，将地址设为 deposit wallet 地址，
// 使得 CreateLevel1Headers/CreateLevel2Headers 中的 POLY_ADDRESS 与订单 signer 一致。
func (s *Signer) SetAddressOverride(addr string) {
	s.addressOverride = addr
}

// GetChainID 返回链ID
func (s *Signer) GetChainID() int {
	return s.chainID
}

// Sign 签名消息哈希
// 对于EIP-712，messageHash已经是最终的哈希值，不需要TextHash
// 对于普通消息签名，应该先使用TextHash
func (s *Signer) Sign(messageHash []byte) (string, error) {
	// 直接签名哈希值（EIP-712已经处理了前缀）
	signature, err := crypto.Sign(messageHash, s.privateKey)
	if err != nil {
		return "", fmt.Errorf("signing failed: %w", err)
	}

	// 添加恢复ID（v = 27 或 28）
	signature[64] += 27

	return hexutil.Encode(signature), nil
}

// GetPrivateKey 返回私钥（用于订单构建器）
func (s *Signer) GetPrivateKey() string {
	keyBytes := crypto.FromECDSA(s.privateKey)
	// 移除0x前缀（如果有）
	keyHex := hexutil.Encode(keyBytes)
	if len(keyHex) > 2 && keyHex[:2] == "0x" {
		return keyHex[2:]
	}
	return keyHex
}
