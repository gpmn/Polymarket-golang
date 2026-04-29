package polymarket

import (
	"encoding/json"
	"fmt"
)

// PostOrder submits an order (v1 or v2).
func (c *ClobClient) PostOrder(order interface{}, orderType OrderType) (*PostOrderResult, error) {
	return c.PostOrderWithOptions(order, orderType, false, false)
}

// PostOrderWithOptions submits an order with post_only and defer_exec options.
func (c *ClobClient) PostOrderWithOptions(order interface{}, orderType OrderType, postOnly bool, deferExec bool) (*PostOrderResult, error) {
	if postOnly && orderType != OrderTypeGTC && orderType != OrderTypeGTD {
		return nil, fmt.Errorf("post_only orders can only be of type GTC or GTD")
	}
	if err := c.assertLevel2Auth(); err != nil {
		return nil, err
	}

	owner := ""
	if c.creds != nil {
		owner = c.creds.APIKey
	}

	var body map[string]interface{}
	switch o := order.(type) {
	case *SignedOrderV2:
		body = OrderToJSONV2(o, owner, orderType, postOnly, deferExec)
	case *SignedOrder:
		body = OrderToJSONV1(o, owner, orderType, postOnly)
	default:
		return nil, fmt.Errorf("unsupported order type: %T", order)
	}

	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal order: %w", err)
	}
	bodyStr := string(bodyJSON)

	requestArgs := &RequestArgs{
		Method:         "POST",
		RequestPath:    PostOrder,
		Body:           body,
		SerializedBody: &bodyStr,
	}

	headers, err := CreateLevel2Headers(c.signer, c.creds, requestArgs)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Post(PostOrder, headers, bodyStr)
	if err != nil {
		return nil, err
	}

	return &PostOrderResult{Payload: body, Response: resp}, nil
}

// PostOrders batch submits orders.
func (c *ClobClient) PostOrders(args []PostOrderArgs) (*PostOrdersResult, error) {
	if err := c.assertLevel2Auth(); err != nil {
		return nil, err
	}

	owner := ""
	if c.creds != nil {
		owner = c.creds.APIKey
	}

	body := make([]map[string]interface{}, len(args))
	for i, arg := range args {
		if arg.PostOnly && arg.OrderType != OrderTypeGTC && arg.OrderType != OrderTypeGTD {
			return nil, fmt.Errorf("post_only orders can only be of type GTC or GTD")
		}
		switch o := arg.Order.(type) {
		case *SignedOrderV2:
			body[i] = OrderToJSONV2(o, owner, arg.OrderType, arg.PostOnly, arg.DeferExec)
		case *SignedOrder:
			body[i] = OrderToJSONV1(o, owner, arg.OrderType, arg.PostOnly)
		default:
			return nil, fmt.Errorf("unsupported order type: %T", arg.Order)
		}
	}

	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal orders: %w", err)
	}
	bodyStr := string(bodyJSON)

	requestArgs := &RequestArgs{
		Method:         "POST",
		RequestPath:    PostOrders,
		Body:           body,
		SerializedBody: &bodyStr,
	}

	headers, err := CreateLevel2Headers(c.signer, c.creds, requestArgs)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Post(PostOrders, headers, bodyStr)
	if err != nil {
		return nil, err
	}

	return &PostOrdersResult{Payload: body, Response: resp}, nil
}

// Cancel cancels an order by orderID.
func (c *ClobClient) Cancel(orderID string) (interface{}, error) {
	if err := c.assertLevel2Auth(); err != nil {
		return nil, err
	}

	body := map[string]string{"orderID": orderID}
	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal cancel request: %w", err)
	}
	bodyStr := string(bodyJSON)

	requestArgs := &RequestArgs{
		Method:         "DELETE",
		RequestPath:    Cancel,
		Body:           body,
		SerializedBody: &bodyStr,
	}

	headers, err := CreateLevel2Headers(c.signer, c.creds, requestArgs)
	if err != nil {
		return nil, err
	}

	return c.httpClient.Delete(Cancel, headers, bodyStr)
}

// CancelOrders cancels multiple orders by order hashes.
func (c *ClobClient) CancelOrders(orderHashes []string) (interface{}, error) {
	if err := c.assertLevel2Auth(); err != nil {
		return nil, err
	}

	bodyJSON, err := json.Marshal(orderHashes)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal order hashes: %w", err)
	}
	bodyStr := string(bodyJSON)

	requestArgs := &RequestArgs{
		Method:         "DELETE",
		RequestPath:    CancelOrders,
		Body:           orderHashes,
		SerializedBody: &bodyStr,
	}

	headers, err := CreateLevel2Headers(c.signer, c.creds, requestArgs)
	if err != nil {
		return nil, err
	}

	return c.httpClient.Delete(CancelOrders, headers, bodyStr)
}

// CancelAll cancels all open orders.
func (c *ClobClient) CancelAll() (interface{}, error) {
	if err := c.assertLevel2Auth(); err != nil {
		return nil, err
	}

	requestArgs := &RequestArgs{Method: "DELETE", RequestPath: CancelAll}
	headers, err := CreateLevel2Headers(c.signer, c.creds, requestArgs)
	if err != nil {
		return nil, err
	}

	return c.httpClient.Delete(CancelAll, headers, nil)
}

// CancelMarketOrders cancels market orders by market/asset_id.
func (c *ClobClient) CancelMarketOrders(params *OrderMarketCancelParams) (interface{}, error) {
	if err := c.assertLevel2Auth(); err != nil {
		return nil, err
	}

	body := map[string]string{}
	if params.Market != "" {
		body["market"] = params.Market
	}
	if params.AssetID != "" {
		body["asset_id"] = params.AssetID
	}

	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal cancel request: %w", err)
	}
	bodyStr := string(bodyJSON)

	requestArgs := &RequestArgs{
		Method:         "DELETE",
		RequestPath:    CancelMarketOrders,
		Body:           body,
		SerializedBody: &bodyStr,
	}

	headers, err := CreateLevel2Headers(c.signer, c.creds, requestArgs)
	if err != nil {
		return nil, err
	}

	return c.httpClient.Delete(CancelMarketOrders, headers, bodyStr)
}

// GetOrders fetches open orders with pagination.
func (c *ClobClient) GetOrders(params *OpenOrderParams, nextCursor string) ([]interface{}, error) {
	if err := c.assertLevel2Auth(); err != nil {
		return nil, err
	}

	if nextCursor == "" {
		nextCursor = INITIAL_CURSOR
	}

	requestArgs := &RequestArgs{Method: "GET", RequestPath: Orders}
	headers, err := CreateLevel2Headers(c.signer, c.creds, requestArgs)
	if err != nil {
		return nil, err
	}

	var results []interface{}
	for nextCursor != EndCursor {
		url := AddQueryOpenOrdersParams(c.host+Orders, params, nextCursor)
		resp, err := c.httpClient.Get(url[len(c.host):], headers)
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

// GetPreMigrationOrders fetches pre-migration (v1) orders.
func (c *ClobClient) GetPreMigrationOrders(nextCursor string) ([]interface{}, error) {
	if err := c.assertLevel2Auth(); err != nil {
		return nil, err
	}

	if nextCursor == "" {
		nextCursor = INITIAL_CURSOR
	}

	requestArgs := &RequestArgs{Method: "GET", RequestPath: PreMigrationOrders}
	headers, err := CreateLevel2Headers(c.signer, c.creds, requestArgs)
	if err != nil {
		return nil, err
	}

	var results []interface{}
	for nextCursor != EndCursor {
		path := fmt.Sprintf("%s?next_cursor=%s", PreMigrationOrders, nextCursor)
		resp, err := c.httpClient.Get(path, headers)
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

// GetOrder fetches a single order by ID.
func (c *ClobClient) GetOrder(orderID string) (interface{}, error) {
	if err := c.assertLevel2Auth(); err != nil {
		return nil, err
	}

	endpoint := GetOrder + orderID
	requestArgs := &RequestArgs{Method: "GET", RequestPath: endpoint}
	headers, err := CreateLevel2Headers(c.signer, c.creds, requestArgs)
	if err != nil {
		return nil, err
	}

	return c.httpClient.Get(endpoint, headers)
}

// GetTrades fetches trade history.
func (c *ClobClient) GetTrades(params *TradeParams, nextCursor string) ([]interface{}, error) {
	if err := c.assertLevel2Auth(); err != nil {
		return nil, err
	}

	if nextCursor == "" {
		nextCursor = INITIAL_CURSOR
	}

	requestArgs := &RequestArgs{Method: "GET", RequestPath: Trades}
	headers, err := CreateLevel2Headers(c.signer, c.creds, requestArgs)
	if err != nil {
		return nil, err
	}

	var results []interface{}
	for nextCursor != EndCursor {
		url := AddQueryTradeParams(c.host+Trades, params, nextCursor)
		resp, err := c.httpClient.Get(url[len(c.host):], headers)
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

// GetBalanceAllowance fetches balance and allowance.
func (c *ClobClient) GetBalanceAllowance(params *BalanceAllowanceParams) (map[string]interface{}, error) {
	if err := c.assertLevel2Auth(); err != nil {
		return nil, err
	}

	if params.SignatureType == nil || (params.SignatureType != nil && *params.SignatureType < 0) {
		if c.builder != nil {
			sigType := c.builder.GetSigType()
			params.SignatureType = &sigType
		} else {
			defaultSigType := 0
			params.SignatureType = &defaultSigType
		}
	}

	url := AddBalanceAllowanceParamsToURL(c.host+GetBalanceAllowance, params)
	requestArgs := &RequestArgs{Method: "GET", RequestPath: GetBalanceAllowance}
	headers, err := CreateLevel2Headers(c.signer, c.creds, requestArgs)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Get(url[len(c.host):], headers)
	if err != nil {
		return nil, err
	}

	respMap, ok := resp.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response format")
	}

	return respMap, nil
}

// GetNotifications fetches notifications.
func (c *ClobClient) GetNotifications() (interface{}, error) {
	if err := c.assertLevel2Auth(); err != nil {
		return nil, err
	}

	sigType := 0
	if c.builder != nil {
		sigType = c.builder.GetSigType()
	}

	url := fmt.Sprintf("%s?signature_type=%d", GetNotifications, sigType)
	requestArgs := &RequestArgs{Method: "GET", RequestPath: GetNotifications}
	headers, err := CreateLevel2Headers(c.signer, c.creds, requestArgs)
	if err != nil {
		return nil, err
	}

	return c.httpClient.Get(url, headers)
}

// DropNotifications deletes notifications.
func (c *ClobClient) DropNotifications(params *DropNotificationParams) (interface{}, error) {
	if err := c.assertLevel2Auth(); err != nil {
		return nil, err
	}

	url := DropNotificationsQueryParams(c.host+DropNotifications, params)
	requestArgs := &RequestArgs{Method: "DELETE", RequestPath: DropNotifications}
	headers, err := CreateLevel2Headers(c.signer, c.creds, requestArgs)
	if err != nil {
		return nil, err
	}

	return c.httpClient.Delete(url[len(c.host):], headers, nil)
}
