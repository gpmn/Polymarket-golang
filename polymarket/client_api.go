package polymarket

import (
	"encoding/json"
	"fmt"
)

// GetOK 健康检查
func (c *ClobClient) GetOK() (interface{}, error) {
	return c.httpClient.Get(OK, nil)
}

// GetServerTime 返回服务器当前时间戳
func (c *ClobClient) GetServerTime() (interface{}, error) {
	return c.httpClient.Get(Time, nil)
}

// GetVersion returns the server API version.
func (c *ClobClient) GetVersion() (int, error) {
	result, err := c.httpClient.Get(VERSION, nil)
	if err != nil {
		return 2, err // default to v2 on error
	}
	if respMap, ok := result.(map[string]interface{}); ok {
		if v, ok := respMap["version"]; ok {
			if vf, ok := v.(float64); ok {
				return int(vf), nil
			}
		}
	}
	return 2, nil
}

// ---- API Key Management ----

// CreateAPIKey 创建新的CLOB API密钥
func (c *ClobClient) CreateAPIKey(nonce *int) (*ApiCreds, error) {
	if err := c.assertLevel1Auth(); err != nil {
		return nil, err
	}
	headers, err := CreateLevel1Headers(c.signer, nonce)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Post(CreateAPIKey, headers, nil)
	if err != nil {
		return nil, err
	}
	respMap, ok := resp.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response format")
	}
	creds := &ApiCreds{
		APIKey:        getStringFromMap(respMap, "apiKey"),
		APISecret:     getStringFromMap(respMap, "secret"),
		APIPassphrase: getStringFromMap(respMap, "passphrase"),
	}
	c.SetAPICreds(creds)
	return creds, nil
}

// DeriveAPIKey 派生已存在的CLOB API密钥
func (c *ClobClient) DeriveAPIKey(nonce *int) (*ApiCreds, error) {
	if err := c.assertLevel1Auth(); err != nil {
		return nil, err
	}
	headers, err := CreateLevel1Headers(c.signer, nonce)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Get(DeriveAPIKey, headers)
	if err != nil {
		return nil, err
	}
	if respArr, ok := resp.([]interface{}); ok {
		if len(respArr) > 0 {
			if respMap, ok := respArr[0].(map[string]interface{}); ok {
				resp = respMap
			}
		}
	}
	respMap, ok := resp.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response format")
	}
	creds := &ApiCreds{
		APIKey:        getStringFromMap(respMap, "apiKey"),
		APISecret:     getStringFromMap(respMap, "secret"),
		APIPassphrase: getStringFromMap(respMap, "passphrase"),
	}
	c.SetAPICreds(creds)
	return creds, nil
}

// CreateOrDeriveAPIKey 创建或派生API凭证
func (c *ClobClient) CreateOrDeriveAPIKey(nonce *int) (*ApiCreds, error) {
	creds, err := c.CreateAPIKey(nonce)
	if err != nil {
		return c.DeriveAPIKey(nonce)
	}
	return creds, nil
}

// GetAPIKeys 获取可用的API密钥列表
func (c *ClobClient) GetAPIKeys() (interface{}, error) {
	if err := c.assertLevel2Auth(); err != nil {
		return nil, err
	}
	requestArgs := &RequestArgs{Method: "GET", RequestPath: GetAPIKeys}
	headers, err := CreateLevel2Headers(c.signer, c.creds, requestArgs)
	if err != nil {
		return nil, err
	}
	return c.httpClient.Get(GetAPIKeys, headers)
}

// GetClosedOnlyMode 获取closed only模式标志
func (c *ClobClient) GetClosedOnlyMode() (interface{}, error) {
	if err := c.assertLevel2Auth(); err != nil {
		return nil, err
	}
	requestArgs := &RequestArgs{Method: "GET", RequestPath: ClosedOnly}
	headers, err := CreateLevel2Headers(c.signer, c.creds, requestArgs)
	if err != nil {
		return nil, err
	}
	return c.httpClient.Get(ClosedOnly, headers)
}

// DeleteAPIKey 删除API密钥
func (c *ClobClient) DeleteAPIKey() (interface{}, error) {
	if err := c.assertLevel2Auth(); err != nil {
		return nil, err
	}
	requestArgs := &RequestArgs{Method: "DELETE", RequestPath: DeleteAPIKey}
	headers, err := CreateLevel2Headers(c.signer, c.creds, requestArgs)
	if err != nil {
		return nil, err
	}
	return c.httpClient.Delete(DeleteAPIKey, headers, nil)
}

// ---- Market data ----

// GetClobMarketInfo fetches CLOB market info and caches tick_size, neg_risk, fee_info.
func (c *ClobClient) GetClobMarketInfo(conditionID string) (map[string]interface{}, error) {
	path := GetClobMarket + conditionID
	resp, err := c.httpClient.Get(path, nil)
	if err != nil {
		return nil, err
	}
	respMap, ok := resp.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response format")
	}

	// Cache market data from response
	if tokens, ok := respMap["t"].([]interface{}); ok {
		for _, t := range tokens {
			if tMap, ok := t.(map[string]interface{}); ok {
				tokenID := fmt.Sprintf("%v", tMap["t"])
				c.mu.Lock()
				c.tokenConditionMap[tokenID] = conditionID
				if mts, ok := respMap["mts"]; ok {
					c.tickSizes[tokenID] = TickSize(fmt.Sprintf("%v", mts))
				}
				if nr, ok := respMap["nr"]; ok {
					if nrBool, ok := nr.(bool); ok {
						c.negRisk[tokenID] = nrBool
					}
				}
				c.mu.Unlock()

				// Cache fee info
				if fd, ok := respMap["fd"].(map[string]interface{}); ok {
					feeInfo := &FeeInfo{}
					if r, ok := fd["r"]; ok {
						if rf, ok := r.(float64); ok {
							feeInfo.Rate = rf
						}
					}
					if e, ok := fd["e"]; ok {
						if ef, ok := e.(float64); ok {
							feeInfo.Exponent = ef
						}
					}
					c.mu.Lock()
					c.feeInfos[tokenID] = feeInfo
					c.mu.Unlock()
				}
			}
		}
	}
	return respMap, nil
}

// GetFeeExponent returns the fee exponent for a token.
func (c *ClobClient) GetFeeExponent(tokenID string) float64 {
	c.mu.RLock()
	if fi, ok := c.feeInfos[tokenID]; ok {
		c.mu.RUnlock()
		return fi.Exponent
	}
	c.mu.RUnlock()
	return 0
}

// GetMidpoint 获取中点价格
func (c *ClobClient) GetMidpoint(tokenID string) (interface{}, error) {
	path := fmt.Sprintf("%s?token_id=%s", MidPoint, tokenID)
	return c.httpClient.Get(path, nil)
}

// GetMidpoints 获取多个token的中点价格
func (c *ClobClient) GetMidpoints(params []BookParams) (interface{}, error) {
	body := make([]map[string]string, len(params))
	for i, p := range params {
		body[i] = map[string]string{"token_id": p.TokenID}
	}
	return c.httpClient.Post(MidPoints, nil, body)
}

// GetPrice 获取市场价格
func (c *ClobClient) GetPrice(tokenID, side string) (interface{}, error) {
	path := fmt.Sprintf("%s?token_id=%s&side=%s", Price, tokenID, side)
	return c.httpClient.Get(path, nil)
}

// GetPrices 获取多个token的市场价格
func (c *ClobClient) GetPrices(params []BookParams) (interface{}, error) {
	body := make([]map[string]string, len(params))
	for i, p := range params {
		body[i] = map[string]string{
			"token_id": p.TokenID,
			"side":     p.Side,
		}
	}
	return c.httpClient.Post(GetPrices, nil, body)
}

// GetPricesHistory 获取价格历史
func (c *ClobClient) GetPricesHistory(params *PricesHistoryParams) (interface{}, error) {
	query := "?"
	if params.Market != "" {
		query += "market=" + params.Market + "&"
	}
	if params.Interval != "" {
		query += "interval=" + params.Interval + "&"
	}
	if params.StartTs > 0 {
		query += fmt.Sprintf("startTs=%d&", params.StartTs)
	}
	if params.EndTs > 0 {
		query += fmt.Sprintf("endTs=%d&", params.EndTs)
	}
	if params.Fidelity > 0 {
		query += fmt.Sprintf("fidelity=%d&", params.Fidelity)
	}
	path := GetPricesHistory + query
	return c.httpClient.Get(path, nil)
}

// GetSpread 获取价差
func (c *ClobClient) GetSpread(tokenID string) (interface{}, error) {
	path := fmt.Sprintf("%s?token_id=%s", GetSpread, tokenID)
	return c.httpClient.Get(path, nil)
}

// GetSpreads 获取多个token的价差
func (c *ClobClient) GetSpreads(params []BookParams) (interface{}, error) {
	body := make([]map[string]string, len(params))
	for i, p := range params {
		body[i] = map[string]string{"token_id": p.TokenID}
	}
	return c.httpClient.Post(GetSpreads, nil, body)
}

// GetTickSize 获取tick size（带缓存）
func (c *ClobClient) GetTickSize(tokenID string) (TickSize, error) {
	c.mu.RLock()
	if tickSize, ok := c.tickSizes[tokenID]; ok {
		c.mu.RUnlock()
		return tickSize, nil
	}
	// Try clob market info cache first
	if condID, ok := c.tokenConditionMap[tokenID]; ok {
		c.mu.RUnlock()
		c.GetClobMarketInfo(condID)
		c.mu.RLock()
		if tickSize, ok := c.tickSizes[tokenID]; ok {
			c.mu.RUnlock()
			return tickSize, nil
		}
		c.mu.RUnlock()
	} else {
		c.mu.RUnlock()
	}

	path := fmt.Sprintf("%s?token_id=%s", GetTickSize, tokenID)
	resp, err := c.httpClient.Get(path, nil)
	if err != nil {
		return "", err
	}
	respMap, ok := resp.(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid response format")
	}
	tickSizeStr := getStringFromMap(respMap, "minimum_tick_size")
	tickSize := TickSize(tickSizeStr)

	c.mu.Lock()
	c.tickSizes[tokenID] = tickSize
	c.mu.Unlock()

	return tickSize, nil
}

// GetNegRisk 获取neg risk标志（带缓存）
func (c *ClobClient) GetNegRisk(tokenID string) (bool, error) {
	c.mu.RLock()
	if negRisk, ok := c.negRisk[tokenID]; ok {
		c.mu.RUnlock()
		return negRisk, nil
	}
	if condID, ok := c.tokenConditionMap[tokenID]; ok {
		c.mu.RUnlock()
		c.GetClobMarketInfo(condID)
		c.mu.RLock()
		if negRisk, ok := c.negRisk[tokenID]; ok {
			c.mu.RUnlock()
			return negRisk, nil
		}
		c.mu.RUnlock()
	} else {
		c.mu.RUnlock()
	}

	path := fmt.Sprintf("%s?token_id=%s", GetNegRisk, tokenID)
	resp, err := c.httpClient.Get(path, nil)
	if err != nil {
		return false, err
	}
	respMap, ok := resp.(map[string]interface{})
	if !ok {
		return false, fmt.Errorf("invalid response format")
	}
	negRisk := getBoolFromMap(respMap, "neg_risk")

	c.mu.Lock()
	c.negRisk[tokenID] = negRisk
	c.mu.Unlock()

	return negRisk, nil
}

// GetFeeRateBps 获取手续费率（基点）（带缓存）
func (c *ClobClient) GetFeeRateBps(tokenID string) (int, error) {
	c.mu.RLock()
	if feeRate, ok := c.feeRates[tokenID]; ok {
		c.mu.RUnlock()
		return feeRate, nil
	}
	c.mu.RUnlock()

	path := fmt.Sprintf("%s?token_id=%s", GetFeeRate, tokenID)
	resp, err := c.httpClient.Get(path, nil)
	if err != nil {
		return 0, err
	}
	respMap, ok := resp.(map[string]interface{})
	if !ok {
		return 0, fmt.Errorf("invalid response format")
	}
	feeRate := 0
	if baseFee, ok := respMap["base_fee"]; ok {
		if feeRateFloat, ok := baseFee.(float64); ok {
			feeRate = int(feeRateFloat)
		}
	}

	c.mu.Lock()
	c.feeRates[tokenID] = feeRate
	c.mu.Unlock()

	return feeRate, nil
}

// GetOrderBook 获取订单簿
func (c *ClobClient) GetOrderBook(tokenID string) (*OrderBookSummary, error) {
	path := fmt.Sprintf("%s?token_id=%s", GetOrderBook, tokenID)
	resp, err := c.httpClient.Get(path, nil)
	if err != nil {
		return nil, err
	}
	respMap, ok := resp.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response format")
	}
	return ParseRawOrderBookSummary(respMap)
}

// GetOrderBooks 获取多个订单簿
func (c *ClobClient) GetOrderBooks(params []BookParams) ([]*OrderBookSummary, error) {
	body := make([]map[string]string, len(params))
	for i, p := range params {
		body[i] = map[string]string{"token_id": p.TokenID}
	}
	resp, err := c.httpClient.Post(GetOrderBooks, nil, body)
	if err != nil {
		return nil, err
	}
	respArray, ok := resp.([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response format")
	}
	orderBooks := make([]*OrderBookSummary, len(respArray))
	for i, item := range respArray {
		if itemMap, ok := item.(map[string]interface{}); ok {
			obs, err := ParseRawOrderBookSummary(itemMap)
			if err != nil {
				return nil, err
			}
			orderBooks[i] = obs
		}
	}
	return orderBooks, nil
}

// GetLastTradePrice 获取最后成交价格
func (c *ClobClient) GetLastTradePrice(tokenID string) (interface{}, error) {
	path := fmt.Sprintf("%s?token_id=%s", GetLastTradePrice, tokenID)
	return c.httpClient.Get(path, nil)
}

// GetLastTradesPrices 获取多个token的最后成交价格
func (c *ClobClient) GetLastTradesPrices(params []BookParams) (interface{}, error) {
	body := make([]map[string]string, len(params))
	for i, p := range params {
		body[i] = map[string]string{"token_id": p.TokenID}
	}
	return c.httpClient.Post(GetLastTradesPrices, nil, body)
}

// ---- Rewards ----

// GetCurrentRewards 获取当前奖励市场列表
func (c *ClobClient) GetCurrentRewards() ([]interface{}, error) {
	var results []interface{}
	nextCursor := INITIAL_CURSOR
	for nextCursor != EndCursor {
		path := fmt.Sprintf("%s?next_cursor=%s", GetRewardsMarketsCurrent, nextCursor)
		resp, err := c.httpClient.Get(path, nil)
		if err != nil {
			return nil, err
		}
		respMap, ok := resp.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid response format")
		}
		if cursor, ok := respMap["next_cursor"].(string); ok {
			nextCursor = cursor
		} else {
			nextCursor = EndCursor
		}
		if data, ok := respMap["data"].([]interface{}); ok {
			results = append(results, data...)
		}
	}
	return results, nil
}

// GetRawRewardsForMarket 获取指定市场的奖励
func (c *ClobClient) GetRawRewardsForMarket(conditionID string) ([]interface{}, error) {
	var results []interface{}
	nextCursor := INITIAL_CURSOR
	for nextCursor != EndCursor {
		path := fmt.Sprintf("%s%s?next_cursor=%s", GetRewardsMarkets, conditionID, nextCursor)
		resp, err := c.httpClient.Get(path, nil)
		if err != nil {
			return nil, err
		}
		respMap, ok := resp.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid response format")
		}
		if cursor, ok := respMap["next_cursor"].(string); ok {
			nextCursor = cursor
		} else {
			nextCursor = EndCursor
		}
		if data, ok := respMap["data"].([]interface{}); ok {
			results = append(results, data...)
		}
	}
	return results, nil
}

// GetEarningsForUserForDay 获取用户每日收益
func (c *ClobClient) GetEarningsForUserForDay(params *EarningsParams) (interface{}, error) {
	if err := c.assertLevel2Auth(); err != nil {
		return nil, err
	}
	requestArgs := &RequestArgs{Method: "GET", RequestPath: GetEarningsForUserForDay}
	headers, err := CreateLevel2Headers(c.signer, c.creds, requestArgs)
	if err != nil {
		return nil, err
	}
	query := "?"
	if params.Date != "" {
		query += "date=" + params.Date + "&"
	}
	if params.Market != "" {
		query += "market=" + params.Market
	}
	return c.httpClient.Get(GetEarningsForUserForDay+query, headers)
}

// GetLiquidityRewardPercentages 获取流动性奖励百分比
func (c *ClobClient) GetLiquidityRewardPercentages() (interface{}, error) {
	if err := c.assertLevel2Auth(); err != nil {
		return nil, err
	}
	requestArgs := &RequestArgs{Method: "GET", RequestPath: GetLiquidityRewardPercentages}
	headers, err := CreateLevel2Headers(c.signer, c.creds, requestArgs)
	if err != nil {
		return nil, err
	}
	return c.httpClient.Get(GetLiquidityRewardPercentages, headers)
}

// GetMarketByToken 根据token ID获取市场
func (c *ClobClient) GetMarketByToken(tokenID string) (interface{}, error) {
	path := GetMarketByToken + tokenID
	return c.httpClient.Get(path, nil)
}

// ---- Builder API keys ----

// CreateBuilderAPIKey creates a new builder API key.
func (c *ClobClient) CreateBuilderAPIKey() (interface{}, error) {
	if err := c.assertLevel2Auth(); err != nil {
		return nil, err
	}
	requestArgs := &RequestArgs{Method: "POST", RequestPath: CreateBuilderAPIKey}
	headers, err := CreateLevel2Headers(c.signer, c.creds, requestArgs)
	if err != nil {
		return nil, err
	}
	return c.httpClient.Post(CreateBuilderAPIKey, headers, nil)
}

// GetBuilderAPIKeys returns the builder API keys.
func (c *ClobClient) GetBuilderAPIKeys() (interface{}, error) {
	if err := c.assertLevel2Auth(); err != nil {
		return nil, err
	}
	requestArgs := &RequestArgs{Method: "GET", RequestPath: GetBuilderAPIKeys}
	headers, err := CreateLevel2Headers(c.signer, c.creds, requestArgs)
	if err != nil {
		return nil, err
	}
	return c.httpClient.Get(GetBuilderAPIKeys, headers)
}

// RevokeBuilderAPIKey revokes a builder API key.
func (c *ClobClient) RevokeBuilderAPIKey(key string) (interface{}, error) {
	if err := c.assertLevel2Auth(); err != nil {
		return nil, err
	}
	body := map[string]string{"key": key}
	bodyJSON, _ := json.Marshal(body)
	bodyStr := string(bodyJSON)
	requestArgs := &RequestArgs{
		Method:         "DELETE",
		RequestPath:    RevokeBuilderAPIKey,
		Body:           body,
		SerializedBody: &bodyStr,
	}
	headers, err := CreateLevel2Headers(c.signer, c.creds, requestArgs)
	if err != nil {
		return nil, err
	}
	return c.httpClient.Delete(RevokeBuilderAPIKey, headers, bodyStr)
}

// GetBuilderFeeRate gets the builder fee rate.
func (c *ClobClient) GetBuilderFeeRate(builderCode string) (interface{}, error) {
	path := GetBuilderFeeRate + builderCode
	return c.httpClient.Get(path, nil)
}

// ---- Helpers ----

func getStringFromMap(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		return fmt.Sprintf("%v", v)
	}
	return ""
}

func getBoolFromMap(m map[string]interface{}, key string) bool {
	if v, ok := m[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}
