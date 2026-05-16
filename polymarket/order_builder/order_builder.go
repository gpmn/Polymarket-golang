package order_builder

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math/big"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/polymarket/go-order-utils/pkg/builder"
	"github.com/polymarket/go-order-utils/pkg/eip712"
	"github.com/polymarket/go-order-utils/pkg/model"
)

// Signer is the signer interface for the order builder.
type Signer interface {
	Address() string
	GetChainID() int
	GetPrivateKey() string
}

// OrderBuilder builds and signs orders for the Polymarket CLOB.
type OrderBuilder struct {
	signer  Signer
	sigType int
	funder  string
}

// NewOrderBuilder creates a new OrderBuilder.
func NewOrderBuilder(s Signer, sigType int, funder string) (*OrderBuilder, error) {
	if s == nil {
		return nil, fmt.Errorf("signer is required")
	}
	if funder == "" {
		funder = s.Address()
	}
	return &OrderBuilder{signer: s, sigType: sigType, funder: funder}, nil
}

// ---- Rounding helpers ----

// RoundConfig defines decimal rounding precision.
type RoundConfig struct {
	Price  int
	Size   int
	Amount int
}

// RoundingConfig maps tick sizes to rounding configurations.
var RoundingConfig = map[string]RoundConfig{
	"0.1":    {Price: 1, Size: 2, Amount: 3},
	"0.01":   {Price: 2, Size: 2, Amount: 4},
	"0.001":  {Price: 3, Size: 2, Amount: 5},
	"0.0001": {Price: 4, Size: 2, Amount: 6},
}

// GetOrderAmounts computes maker/taker amounts for limit orders.
func (ob *OrderBuilder) GetOrderAmounts(side string, size, price float64, roundConfig RoundConfig) (model.Side, *big.Int, *big.Int, error) {
	rawPrice := RoundNormal(price, roundConfig.Price)

	if side == "BUY" {
		rawTakerAmt := RoundDown(size, roundConfig.Size)
		rawMakerAmt := rawTakerAmt * rawPrice
		if DecimalPlaces(rawMakerAmt) > roundConfig.Amount {
			rawMakerAmt = RoundUp(rawMakerAmt, roundConfig.Amount+4)
			if DecimalPlaces(rawMakerAmt) > roundConfig.Amount {
				rawMakerAmt = RoundDown(rawMakerAmt, roundConfig.Amount)
			}
		}
		return model.BUY, big.NewInt(ToTokenDecimals(rawMakerAmt)), big.NewInt(ToTokenDecimals(rawTakerAmt)), nil
	}

	if side == "SELL" {
		rawMakerAmt := RoundDown(size, roundConfig.Size)
		rawTakerAmt := rawMakerAmt * rawPrice
		if DecimalPlaces(rawTakerAmt) > roundConfig.Amount {
			rawTakerAmt = RoundUp(rawTakerAmt, roundConfig.Amount+4)
			if DecimalPlaces(rawTakerAmt) > roundConfig.Amount {
				rawTakerAmt = RoundDown(rawTakerAmt, roundConfig.Amount)
			}
		}
		return model.SELL, big.NewInt(ToTokenDecimals(rawMakerAmt)), big.NewInt(ToTokenDecimals(rawTakerAmt)), nil
	}

	return 0, nil, nil, fmt.Errorf("side must be BUY or SELL")
}

// GetMarketOrderAmounts computes maker/taker amounts for market orders.
// V2: uses RoundDown for price (v1 used RoundNormal).
func (ob *OrderBuilder) GetMarketOrderAmounts(side string, amount, price float64, roundConfig RoundConfig) (model.Side, *big.Int, *big.Int, error) {
	rawPrice := RoundDown(price, roundConfig.Price)

	if side == "BUY" {
		rawMakerAmt := RoundDown(amount, roundConfig.Size)
		rawTakerAmt := rawMakerAmt / rawPrice
		if DecimalPlaces(rawTakerAmt) > roundConfig.Amount {
			rawTakerAmt = RoundUp(rawTakerAmt, roundConfig.Amount+4)
			if DecimalPlaces(rawTakerAmt) > roundConfig.Amount {
				rawTakerAmt = RoundDown(rawTakerAmt, roundConfig.Amount)
			}
		}
		return model.BUY, big.NewInt(ToTokenDecimals(rawMakerAmt)), big.NewInt(ToTokenDecimals(rawTakerAmt)), nil
	}

	if side == "SELL" {
		rawMakerAmt := RoundDown(amount, roundConfig.Size)
		rawTakerAmt := rawMakerAmt * rawPrice
		if DecimalPlaces(rawTakerAmt) > roundConfig.Amount {
			rawTakerAmt = RoundUp(rawTakerAmt, roundConfig.Amount+4)
			if DecimalPlaces(rawTakerAmt) > roundConfig.Amount {
				rawTakerAmt = RoundDown(rawTakerAmt, roundConfig.Amount)
			}
		}
		return model.SELL, big.NewInt(ToTokenDecimals(rawMakerAmt)), big.NewInt(ToTokenDecimals(rawTakerAmt)), nil
	}

	return 0, nil, nil, fmt.Errorf("side must be BUY or SELL")
}

// ---- Market price calculation ----

// OrderSummary is the interface for order book entries.
type OrderSummary interface {
	GetPrice() string
	GetSize() string
}

// CalculateBuyMarketPrice computes the market buy price.
func (ob *OrderBuilder) CalculateBuyMarketPrice(positions []interface{}, amountToMatch float64, orderType string) (float64, error) {
	if len(positions) == 0 {
		return 0, fmt.Errorf("no match")
	}
	sum := 0.0
	for i := len(positions) - 1; i >= 0; i-- {
		pos, ok := positions[i].(OrderSummary)
		if !ok {
			continue
		}
		price, _ := strconv.ParseFloat(pos.GetPrice(), 64)
		size, _ := strconv.ParseFloat(pos.GetSize(), 64)
		sum += size * price
		if sum >= amountToMatch {
			return price, nil
		}
	}
	if orderType == "FOK" {
		return 0, fmt.Errorf("no match")
	}
	if pos, ok := positions[0].(OrderSummary); ok {
		p, _ := strconv.ParseFloat(pos.GetPrice(), 64)
		return p, nil
	}
	return 0, fmt.Errorf("invalid position format")
}

// CalculateSellMarketPrice computes the market sell price.
func (ob *OrderBuilder) CalculateSellMarketPrice(positions []interface{}, amountToMatch float64, orderType string) (float64, error) {
	if len(positions) == 0 {
		return 0, fmt.Errorf("no match")
	}
	sum := 0.0
	for i := len(positions) - 1; i >= 0; i-- {
		pos, ok := positions[i].(OrderSummary)
		if !ok {
			continue
		}
		size, _ := strconv.ParseFloat(pos.GetSize(), 64)
		sum += size
		if sum >= amountToMatch {
			p, _ := strconv.ParseFloat(pos.GetPrice(), 64)
			return p, nil
		}
	}
	if orderType == "FOK" {
		return 0, fmt.Errorf("no match")
	}
	if pos, ok := positions[0].(OrderSummary); ok {
		p, _ := strconv.ParseFloat(pos.GetPrice(), 64)
		return p, nil
	}
	return 0, fmt.Errorf("invalid position format")
}

// ---- V1 order building (delegates to go-order-utils) ----

// BuildSignedOrder builds and signs a v1 order.
func (ob *OrderBuilder) BuildSignedOrder(orderData *model.OrderData, exchangeAddr string, chainID int, negRisk bool) (*model.SignedOrder, error) {
	privateKeyHex := ob.signer.GetPrivateKey()
	if len(privateKeyHex) > 2 && privateKeyHex[:2] == "0x" {
		privateKeyHex = privateKeyHex[2:]
	}
	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %w", err)
	}
	chainIDBig := big.NewInt(int64(chainID))
	orderBuilder := builder.NewExchangeOrderBuilderImpl(chainIDBig, nil)

	var contract model.VerifyingContract
	if negRisk {
		contract = model.NegRiskCTFExchange
	} else {
		contract = model.CTFExchange
	}
	return orderBuilder.BuildSignedOrder(privateKey, orderData, contract)
}

// ---- V2 EIP-712 constants ----

var (
	v2DomainName    = crypto.Keccak256Hash([]byte("Polymarket CTF Exchange"))
	v2DomainVersion = crypto.Keccak256Hash([]byte("2"))

	v2OrderStructHash = crypto.Keccak256Hash([]byte(
		"Order(uint256 salt,address maker,address signer,uint256 tokenId,uint256 makerAmount,uint256 takerAmount,uint8 side,uint8 signatureType,uint256 timestamp,bytes32 metadata,bytes32 builder)",
	))

	v2OrderTypes = []abi.Type{
		eip712.Bytes32, // typehash
		eip712.Uint256, // salt
		eip712.Address, // maker
		eip712.Address, // signer
		eip712.Uint256, // tokenId
		eip712.Uint256, // makerAmount
		eip712.Uint256, // takerAmount
		eip712.Uint8,   // side
		eip712.Uint8,   // signatureType
		eip712.Uint256, // timestamp
		eip712.Bytes32, // metadata
		eip712.Bytes32, // builder
	}

	// ---- POLY_1271 (Deposit Wallet) constants ----
	depositWalletNameHash    = crypto.Keccak256Hash([]byte("DepositWallet"))
	depositWalletVersionHash = crypto.Keccak256Hash([]byte("1"))
	depositWalletDomainSalt  = common.Hash{} // bytes32 zero

	domainTypeHash = crypto.Keccak256Hash([]byte(
		"EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)",
	))

	typedDataSignTypeHash = crypto.Keccak256Hash([]byte(
		"TypedDataSign(Order contents,string name,string version,uint256 chainId,address verifyingContract,bytes32 salt)" +
			"Order(uint256 salt,address maker,address signer,uint256 tokenId,uint256 makerAmount,uint256 takerAmount,uint8 side,uint8 signatureType,uint256 timestamp,bytes32 metadata,bytes32 builder)",
	))

	poly1271ContentsTypes = []abi.Type{
		eip712.Bytes32, // typehash
		eip712.Uint256, // salt
		eip712.Address, // maker
		eip712.Address, // signer
		eip712.Uint256, // tokenId
		eip712.Uint256, // makerAmount
		eip712.Uint256, // takerAmount
		eip712.Uint8,   // side
		eip712.Uint8,   // signatureType
		eip712.Uint256, // timestamp
		eip712.Bytes32, // metadata
		eip712.Bytes32, // builder
	}

	typedDataSignTypes = []abi.Type{
		eip712.Bytes32, // typehash
		eip712.Bytes32, // contents_hash
		eip712.Bytes32, // name_hash
		eip712.Bytes32, // version_hash
		eip712.Uint256, // chainId
		eip712.Address, // verifyingContract
		eip712.Bytes32, // salt
	}
)

// SignedOrderV2Data holds the fields of a v2 signed order.
type SignedOrderV2Data struct {
	Salt          string
	Maker         string
	Signer        string
	TokenId       string
	MakerAmount   string
	TakerAmount   string
	Side          int // 0=BUY, 1=SELL
	Expiration    string
	SignatureType int
	Timestamp     string
	Metadata      string
	Builder       string
	Signature     string
}

// BuildSignedOrderV2 builds and signs a v2 order using EIP-712.
func (ob *OrderBuilder) BuildSignedOrderV2(orderData *SignedOrderV2Data, exchangeAddr string) (*SignedOrderV2Data, error) {
	if orderData.SignatureType == 3 {
		return ob.buildSignedOrderV2Poly1271(orderData, exchangeAddr)
	}

	chainID := big.NewInt(int64(ob.signer.GetChainID()))
	verifyingContract := common.HexToAddress(exchangeAddr)

	domainSeparator, err := eip712.BuildEIP712DomainSeparator(
		v2DomainName, v2DomainVersion, chainID, verifyingContract,
	)
	if err != nil {
		return nil, fmt.Errorf("domain separator: %w", err)
	}

	// Parse big.Int fields
	salt, ok := new(big.Int).SetString(orderData.Salt, 10)
	if !ok {
		return nil, fmt.Errorf("invalid salt: %s", orderData.Salt)
	}
	tokenId, ok := new(big.Int).SetString(orderData.TokenId, 10)
	if !ok {
		return nil, fmt.Errorf("invalid tokenId: %s", orderData.TokenId)
	}
	makerAmount, ok := new(big.Int).SetString(orderData.MakerAmount, 10)
	if !ok {
		return nil, fmt.Errorf("invalid makerAmount: %s", orderData.MakerAmount)
	}
	takerAmount, ok := new(big.Int).SetString(orderData.TakerAmount, 10)
	if !ok {
		return nil, fmt.Errorf("invalid takerAmount: %s", orderData.TakerAmount)
	}
	timestamp, ok := new(big.Int).SetString(orderData.Timestamp, 10)
	if !ok {
		return nil, fmt.Errorf("invalid timestamp: %s", orderData.Timestamp)
	}

	// Parse bytes32 fields (preserve the full 32-byte hex value)
	metadataB32 := hexToBytes32(orderData.Metadata)
	builderB32 := hexToBytes32(orderData.Builder)

	values := []interface{}{
		v2OrderStructHash,
		salt,
		common.HexToAddress(orderData.Maker),
		common.HexToAddress(orderData.Signer),
		tokenId,
		makerAmount,
		takerAmount,
		uint8(orderData.Side),
		uint8(orderData.SignatureType),
		timestamp,
		metadataB32,
		builderB32,
	}

	orderHash, err := eip712.HashTypedDataV4(domainSeparator, v2OrderTypes, values)
	if err != nil {
		return nil, fmt.Errorf("hash typed data: %w", err)
	}

	// Sign
	privateKeyHex := ob.signer.GetPrivateKey()
	if len(privateKeyHex) > 2 && privateKeyHex[:2] == "0x" {
		privateKeyHex = privateKeyHex[2:]
	}
	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return nil, fmt.Errorf("private key: %w", err)
	}
	sig, err := crypto.Sign(orderHash.Bytes(), privateKey)
	if err != nil {
		return nil, fmt.Errorf("sign: %w", err)
	}
	sig[64] += 27 // Ethereum replay protection

	result := *orderData
	result.Signature = "0x" + common.Bytes2Hex(sig)
	return &result, nil
}

// buildSignedOrderV2Poly1271 builds and signs a v2 order for POLY_1271 (Deposit Wallet).
// Uses Solady TypedDataSign wrapping over standard EIP-712 Order signature.
func (ob *OrderBuilder) buildSignedOrderV2Poly1271(orderData *SignedOrderV2Data, exchangeAddr string) (*SignedOrderV2Data, error) {
	chainID := big.NewInt(int64(ob.signer.GetChainID()))
	verifyingContract := common.HexToAddress(exchangeAddr)

	// 1. Compute app_domain_separator (CTF Exchange domain separator)
	appDomainSeparator, err := eip712.BuildEIP712DomainSeparator(
		v2DomainName, v2DomainVersion, chainID, verifyingContract,
	)
	if err != nil {
		return nil, fmt.Errorf("domain separator: %w", err)
	}

	// 2. Parse big.Int fields
	salt, ok := new(big.Int).SetString(orderData.Salt, 10)
	if !ok {
		return nil, fmt.Errorf("invalid salt: %s", orderData.Salt)
	}
	tokenId, ok := new(big.Int).SetString(orderData.TokenId, 10)
	if !ok {
		return nil, fmt.Errorf("invalid tokenId: %s", orderData.TokenId)
	}
	makerAmount, ok := new(big.Int).SetString(orderData.MakerAmount, 10)
	if !ok {
		return nil, fmt.Errorf("invalid makerAmount: %s", orderData.MakerAmount)
	}
	takerAmount, ok := new(big.Int).SetString(orderData.TakerAmount, 10)
	if !ok {
		return nil, fmt.Errorf("invalid takerAmount: %s", orderData.TakerAmount)
	}
	timestamp, ok := new(big.Int).SetString(orderData.Timestamp, 10)
	if !ok {
		return nil, fmt.Errorf("invalid timestamp: %s", orderData.Timestamp)
	}

	metadataB32 := hexToBytes32(orderData.Metadata)
	builderB32 := hexToBytes32(orderData.Builder)

	// 3. Compute contents_hash (Order struct hash)
	contentsValues := []interface{}{
		v2OrderStructHash,
		salt,
		common.HexToAddress(orderData.Maker),
		common.HexToAddress(orderData.Signer),
		tokenId,
		makerAmount,
		takerAmount,
		uint8(orderData.Side),
		uint8(orderData.SignatureType),
		timestamp,
		metadataB32,
		builderB32,
	}
	encodedContents, err := eip712.Encode(poly1271ContentsTypes, contentsValues)
	if err != nil {
		return nil, fmt.Errorf("encode contents: %w", err)
	}
	contentsHash := crypto.Keccak256Hash(encodedContents)

	// 4. Compute typed_data_sign_struct_hash
	typedDataSignValues := []interface{}{
		typedDataSignTypeHash,
		contentsHash,
		depositWalletNameHash,
		depositWalletVersionHash,
		chainID,
		common.HexToAddress(orderData.Signer),
		depositWalletDomainSalt,
	}
	encodedTypedDataSign, err := eip712.Encode(typedDataSignTypes, typedDataSignValues)
	if err != nil {
		return nil, fmt.Errorf("encode typed data sign: %w", err)
	}
	typedDataSignStructHash := crypto.Keccak256Hash(encodedTypedDataSign)

	// 5. Compute digest = keccak256("\x19\x01" + appDomainSeparator + typedDataSignStructHash)
	digest := crypto.Keccak256Hash(
		append(append([]byte("\x19\x01"), appDomainSeparator.Bytes()...), typedDataSignStructHash.Bytes()...),
	)

	// 6. Sign digest (raw ECDSA, no +27)
	privateKeyHex := ob.signer.GetPrivateKey()
	if len(privateKeyHex) > 2 && privateKeyHex[:2] == "0x" {
		privateKeyHex = privateKeyHex[2:]
	}
	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return nil, fmt.Errorf("private key: %w", err)
	}
	innerSig, err := crypto.Sign(digest.Bytes(), privateKey)
	if err != nil {
		return nil, fmt.Errorf("sign: %w", err)
	}
	// Note: no +27 for POLY_1271 inner signature (raw recovery id)

	// 7. Assemble final signature
	// format: inner_signature(65B) + app_domain_separator(32B) + contents_hash(32B) + contents_type + contents_type_len(2B big-endian)
	orderTypeString := "Order(uint256 salt,address maker,address signer,uint256 tokenId,uint256 makerAmount,uint256 takerAmount,uint8 side,uint8 signatureType,uint256 timestamp,bytes32 metadata,bytes32 builder)"
	contentsTypeHex := hex.EncodeToString([]byte(orderTypeString))
	contentsTypeLen := make([]byte, 2)
	binary.BigEndian.PutUint16(contentsTypeLen, uint16(len(orderTypeString)))
	contentsTypeLenHex := hex.EncodeToString(contentsTypeLen)

	finalSig := "0x" + common.Bytes2Hex(innerSig) + appDomainSeparator.Hex()[2:] + contentsHash.Hex()[2:] + contentsTypeHex + contentsTypeLenHex

	result := *orderData
	result.Signature = finalSig
	return &result, nil
}

// hexToBytes32 converts a 0x-prefixed hex string to [32]byte.
func hexToBytes32(hexStr string) [32]byte {
	var result [32]byte
	if len(hexStr) >= 2 && hexStr[:2] == "0x" {
		hexStr = hexStr[2:]
	}
	b := common.FromHex("0x" + hexStr)
	copy(result[:], b[:])
	return result
}

// ---- Accessors ----

// GetSigType returns the signature type.
func (ob *OrderBuilder) GetSigType() int { return ob.sigType }

// GetFunder returns the funder address.
func (ob *OrderBuilder) GetFunder() string { return ob.funder }

// GetSigner returns the signer.
func (ob *OrderBuilder) GetSigner() Signer { return ob.signer }

// GetV2OrderSigner returns the signer address for v2 orders.
// For POLY_1271 (deposit wallet), the signer is the funder (deposit wallet address).
func (ob *OrderBuilder) GetV2OrderSigner() string {
	if ob.sigType == 3 {
		return ob.funder
	}
	return ob.signer.Address()
}

// ---- Timestamp generator ----

// CurrentTimestampMs returns the current Unix time in milliseconds.
func CurrentTimestampMs() string {
	return strconv.FormatInt(time.Now().UnixMilli(), 10)
}

// GenerateSalt returns a random salt as a decimal string (int64-safe, matches JS SDK).
func GenerateSalt() string {
	ms := time.Now().UnixMilli()
	salt := int64(float64(ms) * (float64(time.Now().Nanosecond()%1000000) / 1e6))
	if salt < 1 {
		salt = ms
	}
	return strconv.FormatInt(salt, 10)
}
