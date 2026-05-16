package polymarket

import (
	"github.com/polymarket/go-order-utils/pkg/model"
)

// SignedOrder is a type alias for the v1 signed order from go-order-utils.
type SignedOrder = model.SignedOrder

// ApiCreds API凭证
type ApiCreds struct {
	APIKey        string `json:"apiKey"`
	APISecret     string `json:"secret"`
	APIPassphrase string `json:"passphrase"`
}

// ReadonlyApiKeyResponse 只读API密钥响应
type ReadonlyApiKeyResponse struct {
	APIKey string `json:"apiKey"`
}

// RequestArgs 请求参数
type RequestArgs struct {
	Method         string
	RequestPath    string
	Body           interface{}
	SerializedBody *string
}

// BookParams 订单簿参数
type BookParams struct {
	TokenID string `json:"token_id"`
	Side    string `json:"side,omitempty"`
}

// OrderArgs 限价订单参数 (v2, default)
type OrderArgs struct {
	TokenID         string  `json:"token_id"`     // 条件代币资产ID
	Price           float64 `json:"price"`        // 订单价格
	Size            float64 `json:"size"`         // 条件代币数量
	Side            string  `json:"side"`         // BUY 或 SELL
	Expiration      int     `json:"expiration"`   // 订单过期时间戳, 0 = 无过期
	BuilderCode     string  `json:"builder_code,omitempty"` // Builder code (bytes32) for fee attribution
	Metadata        string  `json:"metadata,omitempty"`     // Optional metadata (bytes32)
	UserUSDCBalance float64 `json:"user_usdc_balance,omitempty"` // 用户pUSD余额, 用于BUY时自动调整订单金额
}

// OrderArgsV1 限价订单参数 (v1 legacy)
type OrderArgsV1 struct {
	TokenID    string  `json:"token_id"`
	Price      float64 `json:"price"`
	Size       float64 `json:"size"`
	Side       string  `json:"side"`
	FeeRateBps int     `json:"fee_rate_bps"`
	Nonce      int     `json:"nonce"`
	Expiration int     `json:"expiration"`
	Taker      string  `json:"taker"`
}

// MarketOrderArgs 市价订单参数 (v2, default)
type MarketOrderArgs struct {
	TokenID         string    `json:"token_id"`         // 条件代币资产ID
	Amount          float64   `json:"amount"`           // BUY: 美元金额, SELL: 份额数量
	Side            string    `json:"side"`             // BUY 或 SELL
	Price           float64   `json:"price"`            // 订单价格（可选, 0=自动计算）
	OrderType       OrderType `json:"order_type"`       // 订单类型
	UserUSDCBalance float64   `json:"user_usdc_balance,omitempty"` // 用户pUSD余额, 用于调整市价买入
	BuilderCode     string    `json:"builder_code,omitempty"`       // Builder code (bytes32)
	Metadata        string    `json:"metadata,omitempty"`           // Optional metadata (bytes32)
}

// MarketOrderArgsV1 市价订单参数 (v1 legacy)
type MarketOrderArgsV1 struct {
	TokenID    string    `json:"token_id"`
	Amount     float64   `json:"amount"`
	Side       string    `json:"side"`
	Price      float64   `json:"price"`
	FeeRateBps int       `json:"fee_rate_bps"`
	Nonce      int       `json:"nonce"`
	Taker      string    `json:"taker"`
	OrderType  OrderType `json:"order_type"`
}

// TradeParams 交易查询参数
type TradeParams struct {
	ID           string `json:"id,omitempty"`
	MakerAddress string `json:"maker_address,omitempty"`
	Market       string `json:"market,omitempty"`
	AssetID      string `json:"asset_id,omitempty"`
	Before       int    `json:"before,omitempty"`
	After        int    `json:"after,omitempty"`
}

// OpenOrderParams 开放订单查询参数
type OpenOrderParams struct {
	ID      string `json:"id,omitempty"`
	Market  string `json:"market,omitempty"`
	AssetID string `json:"asset_id,omitempty"`
}

// DropNotificationParams 删除通知参数
type DropNotificationParams struct {
	IDs []string `json:"ids,omitempty"`
}

// OrderSummary 订单摘要
type OrderSummary struct {
	Price string `json:"price"`
	Size  string `json:"size"`
}

// OrderBookSummary 订单簿摘要
type OrderBookSummary struct {
	Market         string         `json:"market"`
	AssetID        string         `json:"asset_id"`
	Timestamp      string         `json:"timestamp"`
	Bids           []OrderSummary `json:"bids"`
	Asks           []OrderSummary `json:"asks"`
	MinOrderSize   string         `json:"min_order_size"`
	NegRisk        bool           `json:"neg_risk"`
	TickSize       string         `json:"tick_size"`
	LastTradePrice string         `json:"last_trade_price,omitempty"`
	Hash           string         `json:"hash"`
}

// AssetType 资产类型
type AssetType string

const (
	AssetTypeCollateral  AssetType = "COLLATERAL"  // 抵押品（如pUSD）
	AssetTypeConditional AssetType = "CONDITIONAL" // 条件代币
)

// BalanceAllowanceParams 余额和授权查询参数
type BalanceAllowanceParams struct {
	AssetType     AssetType `json:"asset_type,omitempty"`
	TokenID       string    `json:"token_id,omitempty"`
	SignatureType *int      `json:"signature_type,omitempty"`
}

// BalanceAllowanceResponse 余额和授权响应
type BalanceAllowanceResponse struct {
	Balance   string `json:"balance"`
	Allowance string `json:"allowance"`
}

// OrderScoringParams 订单评分参数
type OrderScoringParams struct {
	OrderID string `json:"order_id"`
}

// OrdersScoringParams 多个订单评分参数
type OrdersScoringParams struct {
	OrderIDs []string `json:"order_ids"`
}

// CreateOrderOptions 创建订单选项
type CreateOrderOptions struct {
	TickSize TickSize `json:"tick_size"`
	NegRisk  bool     `json:"neg_risk"`
}

// PartialCreateOrderOptions 部分创建订单选项
type PartialCreateOrderOptions struct {
	TickSize  *TickSize  `json:"tick_size,omitempty"`
	NegRisk   *bool      `json:"neg_risk,omitempty"`
	OrderType *OrderType `json:"order_type,omitempty"`
	PostOnly  *bool      `json:"post_only,omitempty"`
	DeferExec *bool      `json:"defer_exec,omitempty"`
}

// RoundConfig 舍入配置
type RoundConfig struct {
	Price  int // 价格小数位数
	Size   int // 数量小数位数
	Amount int // 金额小数位数
}

// ContractConfig 合约配置 (v1 + v2)
type ContractConfig struct {
	Exchange          string `json:"exchange"`            // V1 exchange
	NegRiskAdapter    string `json:"neg_risk_adapter"`    // V1 neg risk adapter
	NegRiskExchange   string `json:"neg_risk_exchange"`   // V1 neg risk exchange
	Collateral        string `json:"collateral"`          // pUSD collateral token
	ConditionalTokens string `json:"conditional_tokens"`  // ERC1155 conditional tokens
	ExchangeV2        string `json:"exchange_v2"`         // V2 exchange
	NegRiskExchangeV2 string `json:"neg_risk_exchange_v2"` // V2 neg risk exchange
}

// OrderDataV2 用于构建v2订单的输入数据
type OrderDataV2 struct {
	Maker         string // maker地址
	TokenId       string // 代币ID
	MakerAmount   string // maker金额
	TakerAmount   string // taker金额
	Side          string // BUY/SELL (will be converted to model.Side)
	Signer        string // 签名者地址
	SignatureType int    // 签名类型
	Timestamp     string // 时间戳(ms)
	Metadata      string // 元数据(bytes32)
	Builder       string // builder code(bytes32)
	Expiration    string // 过期时间戳
}

// SignedOrderV2 已签名的v2订单 (自包含, 不依赖go-order-utils)
type SignedOrderV2 struct {
	Salt          string `json:"salt"`
	Maker         string `json:"maker"`
	Signer        string `json:"signer"`
	TokenId       string `json:"tokenId"`
	MakerAmount   string `json:"makerAmount"`
	TakerAmount   string `json:"takerAmount"`
	Side          string `json:"side"`          // "BUY" or "SELL" as string
	SideValue     int    `json:"-"`             // 0=BUY, 1=SELL
	Expiration    string `json:"expiration"`
	SignatureType int    `json:"signatureType"` // 0=EOA, 1=PolyProxy, 2=GnosisSafe, 3=Poly1271
	Timestamp     string `json:"timestamp"`
	Metadata      string `json:"metadata"`
	Builder       string `json:"builder"`
	Signature     string `json:"signature"` // 0x-prefixed hex
}

// PostOrderArgs 提交订单参数
type PostOrderArgs struct {
	Order     interface{} `json:"order"`     // *SignedOrderV2 or *model.SignedOrder
	OrderType OrderType   `json:"orderType"`
	DeferExec bool        `json:"deferExec,omitempty"`
	PostOnly  bool        `json:"postOnly,omitempty"`
}

// PostOrderResult 提交订单的结果
type PostOrderResult struct {
	Payload  map[string]interface{} `json:"payload"`
	Response interface{}            `json:"response"`
}

// PostOrdersResult 批量提交订单的结果
type PostOrdersResult struct {
	Payload  []map[string]interface{} `json:"payload"`
	Response interface{}              `json:"response"`
}

// OrderPayload cancel订单的请求体
type OrderPayload struct {
	OrderID string `json:"orderID"`
}

// OrderMarketCancelParams cancel-market-orders的参数
type OrderMarketCancelParams struct {
	Market  string `json:"market,omitempty"`
	AssetID string `json:"asset_id,omitempty"`
}

// BuilderConfig Builder configuration for fee attribution
type BuilderConfig struct {
	BuilderAddress string `json:"builder_address"`
	BuilderCode    string `json:"builder_code"` // bytes32
}

// FeeInfo 费用信息
type FeeInfo struct {
	Rate     float64 `json:"rate"`
	Exponent float64 `json:"exponent"`
}

// FeeDetails 平台费用详情
type FeeDetails struct {
	Rate     float64 `json:"rate"`     // fee rate (e.g. 0.05 for 5%), matching py-clob-client-v2
	Exponent int     `json:"exponent"` // fee exponent
	TakerOnly bool   `json:"taker_only,omitempty"` // if true, fee applies to takers only
}

// ClobToken represents a YES or NO token in a CLOB market.
type ClobToken struct {
	TokenID string `json:"token_id"`
	Outcome string `json:"outcome"`
}

// ClobRewards represents rewards configuration for a market.
type ClobRewards struct {
	MinSize           *float64 `json:"min_size,omitempty"`
	MaxSpread         *float64 `json:"max_spread,omitempty"`
	Enabled           *bool    `json:"enabled,omitempty"`
	SkipMinOrderAge   *bool    `json:"skip_min_order_age,omitempty"`
	MinOrderAgeSeconds *int    `json:"min_order_age_seconds,omitempty"`
}

// MarketDetails represents cached market details from the CLOB API.
type MarketDetails struct {
	ConditionID          string       `json:"condition_id"`
	Tokens               []ClobToken  `json:"tokens,omitempty"`
	MinTickSize          *float64     `json:"min_tick_size,omitempty"`
	NegRisk              *bool        `json:"neg_risk,omitempty"`
	FeeDetails           *FeeDetails  `json:"fee_details,omitempty"`
	MakerBaseFee         *int         `json:"maker_base_fee,omitempty"`
	TakerBaseFee         *int         `json:"taker_base_fee,omitempty"`
	Rewards              *ClobRewards `json:"rewards,omitempty"`
	AcceptingOrders      *bool        `json:"accepting_orders,omitempty"`
	MinOrderSize         *float64     `json:"min_order_size,omitempty"`
	SecondsDelay         *int         `json:"seconds_delay,omitempty"`
	GameStartTime        *string      `json:"game_start_time,omitempty"`
	ClearBookOnStart     *bool        `json:"clear_book_on_start,omitempty"`
	AcceptingOrdersTimestamp *string  `json:"accepting_orders_timestamp,omitempty"`
	RfqEnabled           *bool        `json:"rfq_enabled,omitempty"`
	TakerOrderDelayEnabled *bool      `json:"taker_order_delay_enabled,omitempty"`
	BlockaidCheckEnabled *bool        `json:"blockaid_check_enabled,omitempty"`
}

// BuilderFeeRate builder的maker/taker费率
type BuilderFeeRate struct {
	Maker float64 `json:"maker"`
	Taker float64 `json:"taker"`
}

// BuilderTradeParams Builder交易查询参数
type BuilderTradeParams struct {
	BuilderCode  string `json:"builder_code"`
	ID           string `json:"id,omitempty"`
	MakerAddress string `json:"maker_address,omitempty"`
	Market       string `json:"market,omitempty"`
	AssetID      string `json:"asset_id,omitempty"`
	Before       string `json:"before,omitempty"`
	After        string `json:"after,omitempty"`
}

// PriceHistoryInterval 价格历史查询的间隔类型
type PriceHistoryInterval string

const (
	PriceHistoryMax      PriceHistoryInterval = "max"
	PriceHistoryOneWeek  PriceHistoryInterval = "1w"
	PriceHistoryOneDay   PriceHistoryInterval = "1d"
	PriceHistorySixHours PriceHistoryInterval = "6h"
	PriceHistoryOneHour  PriceHistoryInterval = "1h"
)

// PricesHistoryParams 价格历史查询参数
type PricesHistoryParams struct {
	Market    string `json:"market,omitempty"`
	StartTs   int    `json:"start_ts,omitempty"`
	EndTs     int    `json:"end_ts,omitempty"`
	Fidelity  int    `json:"fidelity,omitempty"`
	Interval  string `json:"interval,omitempty"`
}

// EarningsParams 收益查询参数
type EarningsParams struct {
	Date   string `json:"date,omitempty"`   // YYYY-MM-DD
	Market string `json:"market,omitempty"`
}

// RewardsMarketsParams rewards markets查询参数
type RewardsMarketsParams struct {
	ConditionID string `json:"condition_id,omitempty"`
	NextCursor  string `json:"next_cursor,omitempty"`
}
