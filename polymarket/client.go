package polymarket

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	obuilder "github.com/0xNetuser/Polymarket-golang/polymarket/order_builder"
	"github.com/0xNetuser/Polymarket-golang/polymarket/rfq"
)

// ClobClient CLOB客户端
type ClobClient struct {
	host       string
	chainID    int
	signer     *Signer
	creds      *ApiCreds
	mode       int
	builder    *obuilder.OrderBuilder
	httpClient *HTTPClient

	// Builder configuration
	builderConfig *BuilderConfig

	// Caches
	tickSizes          map[string]TickSize
	negRisk            map[string]bool
	feeRates           map[string]int
	feeInfos           map[string]*FeeInfo
	builderFeeRates    map[string]*BuilderFeeRate
	tokenConditionMap  map[string]string

	// Server version detection
	cachedVersion *int

	// RFQ客户端
	rfq *rfq.RfqClient

	// Fee slippage configuration (0 = no slippage buffer)
	feeSlippage float64

	// Use server time for authentication headers instead of local time
	useServerTime bool

	mu sync.RWMutex
}

// NewClobClient 创建新的CLOB客户端
func NewClobClient(host string, chainID int, privateKey string, creds *ApiCreds, signatureType *int, funder string) (*ClobClient, error) {
	return NewClobClientWithOptions(host, chainID, privateKey, creds, signatureType, funder, nil)
}

// NewClobClientWithOptions creates a CLOB client with optional builder config.
func NewClobClientWithOptions(host string, chainID int, privateKey string, creds *ApiCreds, signatureType *int, funder string, builderConfig *BuilderConfig) (*ClobClient, error) {
	if strings.HasSuffix(host, "/") {
		host = host[:len(host)-1]
	}

	client := &ClobClient{
		host:             host,
		chainID:          chainID,
		creds:            creds,
		httpClient:       NewHTTPClient(host),
		builderConfig:    builderConfig,
		tickSizes:        make(map[string]TickSize),
		negRisk:          make(map[string]bool),
		feeRates:         make(map[string]int),
		feeInfos:         make(map[string]*FeeInfo),
		builderFeeRates:  make(map[string]*BuilderFeeRate),
		tokenConditionMap: make(map[string]string),
	}

	if privateKey != "" {
		signer, err := NewSigner(privateKey, chainID)
		if err != nil {
			return nil, fmt.Errorf("failed to create signer: %w", err)
		}
		client.signer = signer

		sigType := 0
		if signatureType != nil {
			sigType = *signatureType
		}

		funderAddr := signer.Address()
		if funder != "" {
			funderAddr = funder
		}

		builder, err := obuilder.NewOrderBuilder(signer, sigType, funderAddr)
		if err != nil {
			return nil, fmt.Errorf("failed to create order builder: %w", err)
		}
		client.builder = builder
	}

	client.mode = client.getClientMode()
	client.rfq = rfq.NewRfqClient(client)

	return client, nil
}

// getClientMode 获取客户端模式
func (c *ClobClient) getClientMode() int {
	if c.signer == nil {
		return L0
	}
	if c.creds == nil {
		return L1
	}
	return L2
}

// GetAddress 返回签名器的地址
func (c *ClobClient) GetAddress() string {
	if c.signer == nil {
		return ""
	}
	return c.signer.Address()
}

// GetCollateralAddress 返回抵押品代币地址
func (c *ClobClient) GetCollateralAddress() string {
	config := getContractConfig(c.chainID)
	if config != nil {
		return config.Collateral
	}
	return ""
}

// GetConditionalAddress 返回条件代币地址
func (c *ClobClient) GetConditionalAddress() string {
	config := getContractConfig(c.chainID)
	if config != nil {
		return config.ConditionalTokens
	}
	return ""
}

// GetExchangeAddress returns the exchange address for the given negRisk and version.
func (c *ClobClient) GetExchangeAddress(negRisk bool) string {
	v := c.resolveVersion()
	return getExchangeAddress(c.chainID, negRisk, v)
}

// GetExchangeAddressV2 returns the v2 exchange address.
func (c *ClobClient) GetExchangeAddressV2(negRisk bool) string {
	return getExchangeAddress(c.chainID, negRisk, 2)
}

// SetAPICreds 设置API凭证
func (c *ClobClient) SetAPICreds(creds *ApiCreds) {
	c.creds = creds
	c.mode = c.getClientMode()
}

// SetBuilderConfig sets the builder configuration.
func (c *ClobClient) SetBuilderConfig(cfg *BuilderConfig) {
	c.builderConfig = cfg
}

// GetBuilderConfig returns the builder configuration.
func (c *ClobClient) GetBuilderConfig() *BuilderConfig {
	return c.builderConfig
}

// SetFeeSlippage sets the fee slippage percentage (0 or between 1 and 100).
func (c *ClobClient) SetFeeSlippage(feeSlippage float64) error {
	if err := validateFeeSlippage(feeSlippage); err != nil {
		return err
	}
	c.mu.Lock()
	c.feeSlippage = feeSlippage
	c.mu.Unlock()
	return nil
}

// GetFeeSlippage returns the current fee slippage percentage.
func (c *ClobClient) GetFeeSlippage() float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.feeSlippage
}

// SetUseServerTime enables/disables using the server's time for authentication headers.
func (c *ClobClient) SetUseServerTime(use bool) {
	c.mu.Lock()
	c.useServerTime = use
	c.mu.Unlock()
}

// UseServerTime returns whether the client uses server time for authentication.
func (c *ClobClient) UseServerTime() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.useServerTime
}

// SetRetryOnError enables/disables one-time retry on transient POST errors.
func (c *ClobClient) SetRetryOnError(retry bool) {
	c.httpClient.SetRetryOnError(retry)
}

// RetryOnError returns whether retry-on-error is enabled.
func (c *ClobClient) RetryOnError() bool {
	return c.httpClient.RetryOnError()
}

// getTimestamp returns the timestamp to use for authentication.
// If useServerTime is enabled, it fetches from the server's /time endpoint.
// Falls back to local time on failure.
func (c *ClobClient) getTimestamp() int {
	c.mu.RLock()
	useServer := c.useServerTime
	c.mu.RUnlock()

	if !useServer {
		return int(time.Now().Unix())
	}

	result, err := c.httpClient.Get(Time, nil)
	if err != nil {
		return int(time.Now().Unix())
	}
	if respMap, ok := result.(map[string]interface{}); ok {
		if t, ok := respMap["time"].(float64); ok {
			return int(t)
		}
		if t, ok := respMap["timestamp"].(float64); ok {
			return int(t)
		}
	}
	return int(time.Now().Unix())
}

// ensureBuilderFeeRateCached fetches and caches the builder fee rate if not already cached.
func (c *ClobClient) ensureBuilderFeeRateCached(builderCode string) {
	if builderCode == "" || builderCode == BYTES32_ZERO {
		return
	}
	c.mu.RLock()
	_, ok := c.builderFeeRates[builderCode]
	c.mu.RUnlock()
	if ok {
		return
	}

	resp, err := c.GetBuilderFeeRate(builderCode)
	if err != nil {
		return
	}
	if respMap, ok := resp.(map[string]interface{}); ok {
		rate := &BuilderFeeRate{}
		if m, ok := respMap["maker"].(float64); ok {
			rate.Maker = m
		}
		if t, ok := respMap["taker"].(float64); ok {
			rate.Taker = t
		}
		c.mu.Lock()
		c.builderFeeRates[builderCode] = rate
		c.mu.Unlock()
	}
}

// adjustBuyAmountForBalance calculates the fee-adjusted amount for a BUY order.
// It uses cached fee info and builder fee rates. If insufficient balance, the amount is reduced.
func (c *ClobClient) adjustBuyAmountForBalance(tokenID string, amount, price, userUSDCBalance float64, builderCode string) float64 {
	c.ensureBuilderFeeRateCached(builderCode)

	builderTakerFeeRate := 0.0
	c.mu.RLock()
	if rate, ok := c.builderFeeRates[builderCode]; ok {
		builderTakerFeeRate = rate.Taker
	}
	c.mu.RUnlock()

	fi := &FeeInfo{}
	c.mu.RLock()
	if info, ok := c.feeInfos[tokenID]; ok {
		fi = info
	}
	c.mu.RUnlock()

	return AdjustBuyAmountForFees(amount, price, userUSDCBalance, fi.Rate, fi.Exponent, builderTakerFeeRate, c.feeSlippage)
}

// GetBuilder returns the order builder.
func (c *ClobClient) GetBuilder() *obuilder.OrderBuilder {
	return c.builder
}

// resolveVersion detects the server's API version.
func (c *ClobClient) resolveVersion() int {
	c.mu.RLock()
	if c.cachedVersion != nil {
		v := *c.cachedVersion
		c.mu.RUnlock()
		return v
	}
	c.mu.RUnlock()

	version := 2 // default to v2
	result, err := c.httpClient.Get(VERSION, nil)
	if err == nil {
		if respMap, ok := result.(map[string]interface{}); ok {
			if v, ok := respMap["version"]; ok {
				if vf, ok := v.(float64); ok {
					version = int(vf)
				}
			}
		}
	}

	c.mu.Lock()
	c.cachedVersion = &version
	c.mu.Unlock()

	return version
}

// refreshVersion forces a server version re-fetch.
func (c *ClobClient) refreshVersion() {
	c.mu.Lock()
	c.cachedVersion = nil
	c.mu.Unlock()
	c.resolveVersion()
}

// ensureMarketInfoCached ensures the clob market info is cached for the given token_id.
func (c *ClobClient) ensureMarketInfoCached(tokenID string) error {
	c.mu.RLock()
	_, hasTick := c.tickSizes[tokenID]
	_, hasCond := c.tokenConditionMap[tokenID]
	c.mu.RUnlock()

	if hasTick && hasCond {
		return nil
	}

	// If we know the condition_id from token -> condition map, use it
	c.mu.RLock()
	condID, hasCond := c.tokenConditionMap[tokenID]
	c.mu.RUnlock()

	if hasCond {
		_, err := c.GetClobMarketInfo(condID)
		return err
	}

	return nil
}

// assertLevel1Auth 断言需要L1认证
func (c *ClobClient) assertLevel1Auth() error {
	if c.mode < L1 {
		return fmt.Errorf(L1AuthUnavailable)
	}
	return nil
}

// assertLevel2Auth 断言需要L2认证
func (c *ClobClient) assertLevel2Auth() error {
	if c.mode < L2 {
		return fmt.Errorf(L2AuthUnavailable)
	}
	return nil
}

// AssertLevel2Auth 断言需要L2认证（导出方法，供RFQ客户端使用）
func (c *ClobClient) AssertLevel2Auth() error {
	return c.assertLevel2Auth()
}

// GetSigner 获取签名器（供RFQ客户端使用）
func (c *ClobClient) GetSigner() *Signer {
	return c.signer
}

// GetCreds 获取API凭证（供RFQ客户端使用）
func (c *ClobClient) GetCreds() *ApiCreds {
	return c.creds
}

// GetHTTPClient 获取HTTP客户端（供RFQ客户端使用）
func (c *ClobClient) GetHTTPClient() rfq.HTTPClientInterface {
	return c.httpClient
}

// GetRFQ 获取RFQ客户端
func (c *ClobClient) GetRFQ() *rfq.RfqClient {
	return c.rfq
}

// CreateLevel2HeadersInternal 创建L2认证头（供RFQ客户端使用）
func (c *ClobClient) CreateLevel2HeadersInternal(method, path string, body interface{}) (map[string]string, error) {
	var bodyStr string
	if body != nil {
		bodyJSON, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal body: %w", err)
		}
		bodyStr = string(bodyJSON)
	}

	requestArgs := &RequestArgs{
		Method:         method,
		RequestPath:    path,
		Body:           body,
		SerializedBody: &bodyStr,
	}

	return CreateLevel2Headers(c.signer, c.creds, requestArgs, c.getTimestamp())
}

// GetHost 获取host（供RFQ客户端使用）
func (c *ClobClient) GetHost() string {
	return c.host
}

// GetAPICreds 获取API Key（供RFQ客户端使用）
func (c *ClobClient) GetAPICreds() string {
	if c.creds != nil {
		return c.creds.APIKey
	}
	return ""
}

// CreateOrderForRFQ 为RFQ创建签名订单（供RFQ客户端使用）
func (c *ClobClient) CreateOrderForRFQ(args *rfq.OrderCreationArgs) (*rfq.SignedOrderData, error) {
	orderArgs := &OrderArgs{
		TokenID:    args.TokenID,
		Price:      args.Price,
		Size:       args.Size,
		Side:       args.Side,
		Expiration: args.Expiration,
	}

	signedOrder, err := c.CreateOrder(orderArgs, nil)
	if err != nil {
		return nil, err
	}

	signedOrderV2, ok := signedOrder.(*SignedOrderV2)
	if !ok {
		return nil, fmt.Errorf("expected v2 signed order")
	}

	sideStr := signedOrderV2.Side
	if sideStr == "" {
		if signedOrderV2.SideValue == 1 {
			sideStr = "SELL"
		} else {
			sideStr = "BUY"
		}
	}

	return &rfq.SignedOrderData{
		Salt:          parseBigIntOrZero(signedOrderV2.Salt),
		Maker:         signedOrderV2.Maker,
		Signer:        signedOrderV2.Signer,
		TokenID:       signedOrderV2.TokenId,
		MakerAmount:   signedOrderV2.MakerAmount,
		TakerAmount:   signedOrderV2.TakerAmount,
		Expiration:    signedOrderV2.Expiration,
		Side:          sideStr,
		SignatureType: signedOrderV2.SignatureType,
		Timestamp:     signedOrderV2.Timestamp,
		Metadata:      signedOrderV2.Metadata,
		Builder:       signedOrderV2.Builder,
		Signature:     signedOrderV2.Signature,
	}, nil
}
