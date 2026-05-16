package polymarket

import (
	"fmt"
	"strconv"

	obuilder "github.com/0xNetuser/Polymarket-golang/polymarket/order_builder"
	"github.com/polymarket/go-order-utils/pkg/model"
)

// resolveTickSize resolves the tick size for a token.
func (c *ClobClient) resolveTickSize(tokenID string, tickSize *TickSize) (TickSize, error) {
	minTickSize, err := c.GetTickSize(tokenID)
	if err != nil {
		return "", err
	}
	if tickSize != nil {
		if IsTickSizeSmaller(*tickSize, minTickSize) {
			return "", fmt.Errorf("invalid tick size (%s), minimum for the market is %s", *tickSize, minTickSize)
		}
		return *tickSize, nil
	}
	return minTickSize, nil
}

// resolveFeeRate resolves fee rate (only used for v1 orders).
func (c *ClobClient) resolveFeeRate(tokenID string, userFeeRate int) (int, error) {
	marketFeeRateBps, err := c.GetFeeRateBps(tokenID)
	if err != nil {
		return 0, err
	}
	if marketFeeRateBps > 0 && userFeeRate > 0 && userFeeRate != marketFeeRateBps {
		return 0, fmt.Errorf("invalid user provided fee rate: (%d), fee rate for the market must be %d", userFeeRate, marketFeeRateBps)
	}
	return marketFeeRateBps, nil
}

// CreateOrder creates and signs a limit order (v2 by default).
func (c *ClobClient) CreateOrder(orderArgs *OrderArgs, options *PartialCreateOrderOptions) (interface{}, error) {
	if err := c.assertLevel1Auth(); err != nil {
		return nil, err
	}

	tokenID := orderArgs.TokenID
	c.ensureMarketInfoCached(tokenID)

	// Resolve tick size
	var tickSizePtr *TickSize
	if options != nil && options.TickSize != nil {
		tickSizePtr = options.TickSize
	}
	tickSize, err := c.resolveTickSize(tokenID, tickSizePtr)
	if err != nil {
		return nil, err
	}

	// Validate price
	if !PriceValid(orderArgs.Price, tickSize) {
		tickSizeFloat, _ := strconv.ParseFloat(string(tickSize), 64)
		return nil, fmt.Errorf("price (%.6f), min: %s - max: %.6f", orderArgs.Price, tickSize, 1.0-tickSizeFloat)
	}

	// Resolve neg risk
	negRisk := false
	if options != nil && options.NegRisk != nil {
		negRisk = *options.NegRisk
	} else {
		negRisk, err = c.GetNegRisk(tokenID)
		if err != nil {
			return nil, err
		}
	}

	// Get rounding config
	roundConfig, ok := obuilder.RoundingConfig[string(tickSize)]
	if !ok {
		return nil, fmt.Errorf("unsupported tick size: %s", tickSize)
	}

	// Round price to tick size (ensures fee adjustment uses the same price as order building)
	roundedPrice := obuilder.RoundNormal(orderArgs.Price, roundConfig.Price)

	// Apply builder code and metadata early (needed for fee adjustment)
	builderCode := orderArgs.BuilderCode
	if builderCode == "" || builderCode == BYTES32_ZERO {
		if c.builderConfig != nil && c.builderConfig.BuilderCode != "" {
			builderCode = c.builderConfig.BuilderCode
		}
	}
	if builderCode == "" {
		builderCode = BYTES32_ZERO
	}

	metadata := orderArgs.Metadata
	if metadata == "" {
		metadata = BYTES32_ZERO
	}

	expiration := "0"
	if orderArgs.Expiration > 0 {
		expiration = strconv.Itoa(orderArgs.Expiration)
	}

	version := c.resolveVersion()

	// Compute order amounts (with fee adjustment for v2 BUY orders when balance is provided)
	size := orderArgs.Size
	if version >= 2 && orderArgs.Side == BUY && orderArgs.UserUSDCBalance > 0 {
		notional := size * roundedPrice
		adjustedNotional := c.adjustBuyAmountForBalance(tokenID, notional, roundedPrice, orderArgs.UserUSDCBalance, builderCode)
		size = adjustedNotional / roundedPrice
	}

	side, makerAmount, takerAmount, err := c.builder.GetOrderAmounts(orderArgs.Side, size, roundedPrice, roundConfig)
	if err != nil {
		return nil, err
	}

	if version >= 2 {
		// Build v2 order
		sideVal := 0
		if side == model.SELL {
			sideVal = 1
		}

		saltStr := obuilder.GenerateSalt()
		timestamp := obuilder.CurrentTimestampMs()

		orderData := &obuilder.SignedOrderV2Data{
			Salt:          saltStr,
			Maker:         c.builder.GetFunder(),
			Signer:        c.builder.GetV2OrderSigner(),
			TokenId:       tokenID,
			MakerAmount:   makerAmount.String(),
			TakerAmount:   takerAmount.String(),
			Side:          sideVal,
			Expiration:    expiration,
			SignatureType: c.builder.GetSigType(),
			Timestamp:     timestamp,
			Metadata:      metadata,
			Builder:       builderCode,
		}

		exchangeAddr := c.GetExchangeAddressV2(negRisk)
		signedOrder, err := c.builder.BuildSignedOrderV2(orderData, exchangeAddr)
		if err != nil {
			return nil, fmt.Errorf("failed to build v2 signed order: %w", err)
		}

		// Convert to public SignedOrderV2 type
		sideStr := "BUY"
		if sideVal == 1 {
			sideStr = "SELL"
		}

		return &SignedOrderV2{
			Salt:          signedOrder.Salt,
			Maker:         signedOrder.Maker,
			Signer:        signedOrder.Signer,
			TokenId:       signedOrder.TokenId,
			MakerAmount:   signedOrder.MakerAmount,
			TakerAmount:   signedOrder.TakerAmount,
			Side:          sideStr,
			SideValue:     sideVal,
			Expiration:    signedOrder.Expiration,
			SignatureType: signedOrder.SignatureType,
			Timestamp:     signedOrder.Timestamp,
			Metadata:      signedOrder.Metadata,
			Builder:       signedOrder.Builder,
			Signature:     signedOrder.Signature,
		}, nil
	}

	// V1 fallback
	feeRateBps, err := c.resolveFeeRate(tokenID, 0)
	if err != nil {
		return nil, err
	}

	taker := ""
	if v1Args, ok := interface{}(orderArgs).(*OrderArgsV1); ok {
		taker = v1Args.Taker
	}
	if taker == "" {
		taker = ZeroAddress
	}

	orderData := &model.OrderData{
		Maker:         c.builder.GetFunder(),
		Taker:         taker,
		TokenId:       tokenID,
		MakerAmount:   makerAmount.String(),
		TakerAmount:   takerAmount.String(),
		Side:          side,
		FeeRateBps:    strconv.Itoa(feeRateBps),
		Nonce:         strconv.Itoa(orderArgs.Nonce()),
		Signer:        c.signer.Address(), // V1 fallback（死代码，服务器版本≥2 走 V2 路径，不执行此代码）
		Expiration:    expiration,
		SignatureType: model.SignatureType(c.builder.GetSigType()),
	}

	contractConfig := getContractConfig(c.chainID)
	var exchangeAddr string
	if negRisk {
		exchangeAddr = contractConfig.NegRiskExchange
	} else {
		exchangeAddr = contractConfig.Exchange
	}

	return c.builder.BuildSignedOrder(orderData, exchangeAddr, c.chainID, negRisk)
}

// CreateMarketOrder creates and signs a market order (v2 by default).
func (c *ClobClient) CreateMarketOrder(orderArgs *MarketOrderArgs, options *PartialCreateOrderOptions) (interface{}, error) {
	if err := c.assertLevel1Auth(); err != nil {
		return nil, err
	}

	tokenID := orderArgs.TokenID
	c.ensureMarketInfoCached(tokenID)

	// Resolve tick size
	var tickSizePtr *TickSize
	if options != nil && options.TickSize != nil {
		tickSizePtr = options.TickSize
	}
	tickSize, err := c.resolveTickSize(tokenID, tickSizePtr)
	if err != nil {
		return nil, err
	}

	// Calculate market price if not set
	if orderArgs.Price <= 0 {
		price, err := c.CalculateMarketPrice(tokenID, orderArgs.Side, orderArgs.Amount, orderArgs.OrderType)
		if err != nil {
			return nil, err
		}
		orderArgs.Price = price
	}

	// Validate price
	if !PriceValid(orderArgs.Price, tickSize) {
		tickSizeFloat, _ := strconv.ParseFloat(string(tickSize), 64)
		return nil, fmt.Errorf("price (%.6f), min: %s - max: %.6f", orderArgs.Price, tickSize, 1.0-tickSizeFloat)
	}

	// Resolve neg risk
	negRisk := false
	if options != nil && options.NegRisk != nil {
		negRisk = *options.NegRisk
	} else {
		negRisk, err = c.GetNegRisk(tokenID)
		if err != nil {
			return nil, err
		}
	}

	// Get rounding config
	roundConfig, ok := obuilder.RoundingConfig[string(tickSize)]
	if !ok {
		return nil, fmt.Errorf("unsupported tick size: %s", tickSize)
	}

	// Apply builder code and metadata early (needed for fee adjustment)
	builderCode := orderArgs.BuilderCode
	if builderCode == "" || builderCode == BYTES32_ZERO {
		if c.builderConfig != nil && c.builderConfig.BuilderCode != "" {
			builderCode = c.builderConfig.BuilderCode
		}
	}
	if builderCode == "" {
		builderCode = BYTES32_ZERO
	}

	metadata := orderArgs.Metadata
	if metadata == "" {
		metadata = BYTES32_ZERO
	}

	version := c.resolveVersion()

	// Round price using RoundDown for market orders (matching py-clob-client-v2 behavior)
	rawPrice := obuilder.RoundDown(orderArgs.Price, roundConfig.Price)

	// Compute order amounts (v2 uses round_down for price in market orders)
	// Fee adjustment for v2 BUY orders when balance is provided
	// Use rounded price for fee calculation to match Python SDK v1.0.1 behavior
	amount := orderArgs.Amount
	if version >= 2 && orderArgs.Side == BUY && orderArgs.UserUSDCBalance > 0 {
		amount = c.adjustBuyAmountForBalance(tokenID, amount, rawPrice, orderArgs.UserUSDCBalance, builderCode)
	}

	side, makerAmount, takerAmount, err := c.builder.GetMarketOrderAmounts(orderArgs.Side, amount, rawPrice, roundConfig)
	if err != nil {
		return nil, err
	}

	if version >= 2 {
		sideVal := 0
		if side == model.SELL {
			sideVal = 1
		}

		saltStr := obuilder.GenerateSalt()
		timestamp := obuilder.CurrentTimestampMs()

		orderData := &obuilder.SignedOrderV2Data{
			Salt:          saltStr,
			Maker:         c.builder.GetFunder(),
			Signer:        c.builder.GetV2OrderSigner(),
			TokenId:       tokenID,
			MakerAmount:   makerAmount.String(),
			TakerAmount:   takerAmount.String(),
			Side:          sideVal,
			Expiration:    "0", // market orders have no expiration
			SignatureType: c.builder.GetSigType(),
			Timestamp:     timestamp,
			Metadata:      metadata,
			Builder:       builderCode,
		}

		exchangeAddr := c.GetExchangeAddressV2(negRisk)
		signedOrder, err := c.builder.BuildSignedOrderV2(orderData, exchangeAddr)
		if err != nil {
			return nil, fmt.Errorf("failed to build v2 market order: %w", err)
		}

		sideStr := "BUY"
		if sideVal == 1 {
			sideStr = "SELL"
		}

		return &SignedOrderV2{
			Salt:          signedOrder.Salt,
			Maker:         signedOrder.Maker,
			Signer:        signedOrder.Signer,
			TokenId:       signedOrder.TokenId,
			MakerAmount:   signedOrder.MakerAmount,
			TakerAmount:   signedOrder.TakerAmount,
			Side:          sideStr,
			SideValue:     sideVal,
			Expiration:    signedOrder.Expiration,
			SignatureType: signedOrder.SignatureType,
			Timestamp:     signedOrder.Timestamp,
			Metadata:      signedOrder.Metadata,
			Builder:       signedOrder.Builder,
			Signature:     signedOrder.Signature,
		}, nil
	}

	// V1 fallback
	feeRateBps, err := c.resolveFeeRate(tokenID, orderArgs.FeeRateBps())
	if err != nil {
		return nil, err
	}

	taker := ""
	if v1Args, ok := interface{}(orderArgs).(*MarketOrderArgsV1); ok {
		taker = v1Args.Taker
	}
	if taker == "" {
		taker = ZeroAddress
	}

	orderData := &model.OrderData{
		Maker:         c.builder.GetFunder(),
		Taker:         taker,
		TokenId:       tokenID,
		MakerAmount:   makerAmount.String(),
		TakerAmount:   takerAmount.String(),
		Side:          side,
		FeeRateBps:    strconv.Itoa(feeRateBps),
		Nonce:         strconv.Itoa(orderArgs.Nonce()),
		Signer:        c.signer.Address(), // V1 fallback（死代码，服务器版本≥2 走 V2 路径，不执行此代码）
		Expiration:    "0",
		SignatureType: model.SignatureType(c.builder.GetSigType()),
	}

	contractConfig := getContractConfig(c.chainID)
	var exchangeAddr string
	if negRisk {
		exchangeAddr = contractConfig.NegRiskExchange
	} else {
		exchangeAddr = contractConfig.Exchange
	}

	return c.builder.BuildSignedOrder(orderData, exchangeAddr, c.chainID, negRisk)
}

// CreateAndPostOrder creates and posts a limit order.
func (c *ClobClient) CreateAndPostOrder(orderArgs *OrderArgs, options *PartialCreateOrderOptions) (*PostOrderResult, error) {
	order, err := c.CreateOrder(orderArgs, options)
	if err != nil {
		return nil, err
	}

	orderType := OrderTypeGTC
	if options != nil && options.OrderType != nil {
		orderType = *options.OrderType
	}

	postOnly := false
	if options != nil && options.PostOnly != nil {
		postOnly = *options.PostOnly
	}

	deferExec := false
	if options != nil && options.DeferExec != nil {
		deferExec = *options.DeferExec
	}

	return c.PostOrderWithOptions(order, orderType, postOnly, deferExec)
}

// CreateAndPostMarketOrder creates and posts a market order.
// Note: post_only is not supported for market orders (FOK/FAK), so it is always false.
func (c *ClobClient) CreateAndPostMarketOrder(orderArgs *MarketOrderArgs, options *PartialCreateOrderOptions) (*PostOrderResult, error) {
	order, err := c.CreateMarketOrder(orderArgs, options)
	if err != nil {
		return nil, err
	}

	orderType := OrderTypeFOK
	if options != nil && options.OrderType != nil {
		orderType = *options.OrderType
	}

	deferExec := false
	if options != nil && options.DeferExec != nil {
		deferExec = *options.DeferExec
	}

	return c.PostOrderWithOptions(order, orderType, false, deferExec)
}

// CalculateMarketPrice computes the market price from the order book.
func (c *ClobClient) CalculateMarketPrice(tokenID, side string, amount float64, orderType OrderType) (float64, error) {
	book, err := c.GetOrderBook(tokenID)
	if err != nil {
		return 0, fmt.Errorf("no orderbook: %w", err)
	}

	if side == BUY {
		if len(book.Asks) == 0 {
			return 0, fmt.Errorf("no match")
		}
		return c.builder.CalculateBuyMarketPrice(ConvertOrderSummaries(book.Asks), amount, string(orderType))
	}

	if len(book.Bids) == 0 {
		return 0, fmt.Errorf("no match")
	}
	return c.builder.CalculateSellMarketPrice(ConvertOrderSummaries(book.Bids), amount, string(orderType))
}

// ConvertOrderSummaries converts OrderSummary to interface{} slice.
func ConvertOrderSummaries(summaries []OrderSummary) []interface{} {
	result := make([]interface{}, len(summaries))
	for i, s := range summaries {
		result[i] = &OrderSummaryWrapper{OrderSummary: s}
	}
	return result
}

// ---- Accessor methods for back-compat with v1 args (used by v1 fallback) ----

// FeeRateBps returns the fee rate (0 for v2).
func (a *OrderArgs) FeeRateBps() int { return 0 }

// Nonce returns the nonce (0 for v2).
func (a *OrderArgs) Nonce() int { return 0 }

// FeeRateBps returns the fee rate (0 for v2).
func (a *MarketOrderArgs) FeeRateBps() int { return 0 }

// Nonce returns the nonce (0 for v2).
func (a *MarketOrderArgs) Nonce() int { return 0 }
