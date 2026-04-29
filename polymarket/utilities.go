package polymarket

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

// ParseRawOrderBookSummary 解析原始订单簿摘要
func ParseRawOrderBookSummary(rawObs map[string]interface{}) (*OrderBookSummary, error) {
	bids := []OrderSummary{}
	if bidsRaw, ok := rawObs["bids"].([]interface{}); ok {
		for _, bidRaw := range bidsRaw {
			if bid, ok := bidRaw.(map[string]interface{}); ok {
				bids = append(bids, OrderSummary{
					Price: fmt.Sprintf("%v", bid["price"]),
					Size:  fmt.Sprintf("%v", bid["size"]),
				})
			}
		}
	}

	asks := []OrderSummary{}
	if asksRaw, ok := rawObs["asks"].([]interface{}); ok {
		for _, askRaw := range asksRaw {
			if ask, ok := askRaw.(map[string]interface{}); ok {
				asks = append(asks, OrderSummary{
					Price: fmt.Sprintf("%v", ask["price"]),
					Size:  fmt.Sprintf("%v", ask["size"]),
				})
			}
		}
	}

	obs := &OrderBookSummary{
		Market:         getString(rawObs, "market"),
		AssetID:        getString(rawObs, "asset_id"),
		Timestamp:      getString(rawObs, "timestamp"),
		MinOrderSize:   getString(rawObs, "min_order_size"),
		NegRisk:        getBool(rawObs, "neg_risk"),
		TickSize:       getString(rawObs, "tick_size"),
		Bids:           bids,
		Asks:           asks,
		LastTradePrice: getString(rawObs, "last_trade_price"),
		Hash:           getString(rawObs, "hash"),
	}

	return obs, nil
}

// GenerateOrderBookSummaryHash 生成订单簿摘要哈希
func GenerateOrderBookSummaryHash(orderbook *OrderBookSummary) string {
	originalHash := orderbook.Hash
	orderbook.Hash = ""

	jsonData, err := json.Marshal(orderbook)
	if err != nil {
		orderbook.Hash = originalHash
		return ""
	}

	hash := sha1.Sum(jsonData)
	hashStr := fmt.Sprintf("%x", hash)

	orderbook.Hash = hashStr
	return hashStr
}

// OrderToJSONV2 converts a v2 signed order to API JSON format.
func OrderToJSONV2(order *SignedOrderV2, owner string, orderType OrderType, postOnly bool, deferExec bool) map[string]interface{} {
	sideStr := order.Side
	if sideStr == "" {
		if order.SideValue == 1 {
			sideStr = "SELL"
		} else {
			sideStr = "BUY"
		}
	}

	saltStr := order.Salt
	if saltStr == "" {
		saltStr = "0"
	}
	tsStr := order.Timestamp
	if tsStr == "" {
		tsStr = "0"
	}

	orderDict := map[string]interface{}{
		"salt":          saltStr,
		"maker":         common.HexToAddress(order.Maker).Hex(),
		"signer":        common.HexToAddress(order.Signer).Hex(),
		"tokenId":       order.TokenId,
		"makerAmount":   order.MakerAmount,
		"takerAmount":   order.TakerAmount,
		"side":          sideStr,
		"expiration":    order.Expiration,
		"signatureType": order.SignatureType,
		"timestamp":     tsStr,
		"metadata":      order.Metadata,
		"builder":       order.Builder,
		"signature":     order.Signature,
	}

	return map[string]interface{}{
		"order":     orderDict,
		"owner":     owner,
		"orderType": string(orderType),
		"postOnly":  postOnly,
		"deferExec": deferExec,
	}
}

// OrderToJSONV1 converts a v1 signed order to API JSON format.
func OrderToJSONV1(order *SignedOrder, owner string, orderType OrderType, postOnly bool) map[string]interface{} {
	var signatureHex string
	if order.Signature != nil {
		sigStr := string(order.Signature)
		if strings.HasPrefix(sigStr, "0x") {
			signatureHex = sigStr
		} else {
			decoded, err := base64.StdEncoding.DecodeString(sigStr)
			if err == nil {
				signatureHex = "0x" + hex.EncodeToString(decoded)
			} else {
				signatureHex = "0x" + hex.EncodeToString(order.Signature)
			}
		}
	}

	makerAddr := common.HexToAddress(order.Maker.Hex())
	takerAddr := common.HexToAddress(order.Taker.Hex())
	signerAddr := common.HexToAddress(order.Signer.Hex())

	sideStr := "BUY"
	if order.Side.Int64() == 1 {
		sideStr = "SELL"
	}

	orderDict := map[string]interface{}{
		"salt":          order.Salt.Int64(),
		"maker":         makerAddr.Hex(),
		"signer":        signerAddr.Hex(),
		"taker":         takerAddr.Hex(),
		"tokenId":       order.TokenId.String(),
		"makerAmount":   order.MakerAmount.String(),
		"takerAmount":   order.TakerAmount.String(),
		"expiration":    order.Expiration.String(),
		"nonce":         order.Nonce.String(),
		"feeRateBps":    order.FeeRateBps.String(),
		"side":          sideStr,
		"signatureType": int(order.SignatureType.Int64()),
		"signature":     signatureHex,
	}
	return map[string]interface{}{
		"order":     orderDict,
		"owner":     owner,
		"orderType": string(orderType),
		"postOnly":  postOnly,
	}
}

// OrderToJSON is the legacy wrapper; prefers v2 format.
// Deprecated: use OrderToJSONV2 directly.
func OrderToJSON(order *SignedOrder, owner string, orderType OrderType) map[string]interface{} {
	return OrderToJSONV1(order, owner, orderType, false)
}

// OrderToJSONWithPostOnly is the legacy wrapper for post_only orders.
// Deprecated: use OrderToJSONV1 directly.
func OrderToJSONWithPostOnly(order *SignedOrder, owner string, orderType OrderType, postOnly bool) map[string]interface{} {
	return OrderToJSONV1(order, owner, orderType, postOnly)
}

// IsV2Order returns true if the order is a v2 signed order.
func IsV2Order(order interface{}) bool {
	_, ok := order.(*SignedOrderV2)
	return ok
}

// IsTickSizeSmaller 检查tick size是否更小
func IsTickSizeSmaller(a, b TickSize) bool {
	aFloat, _ := strconv.ParseFloat(string(a), 64)
	bFloat, _ := strconv.ParseFloat(string(b), 64)
	return aFloat < bFloat
}

// PriceValid 检查价格是否有效
func PriceValid(price float64, tickSize TickSize) bool {
	tickSizeFloat, _ := strconv.ParseFloat(string(tickSize), 64)
	return price >= tickSizeFloat && price <= 1.0-tickSizeFloat
}

// parseBigIntOrZero parses a string as a big integer, returning 0 on failure.
func parseBigIntOrZero(s string) int64 {
	if s == "" {
		return 0
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0
	}
	return n
}

// 辅助函数
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		return fmt.Sprintf("%v", v)
	}
	return ""
}

func getBool(m map[string]interface{}, key string) bool {
	if v, ok := m[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}
