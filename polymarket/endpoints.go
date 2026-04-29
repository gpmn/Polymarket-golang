package polymarket

// API端点常量
const (
	OK      = "/ok"
	Time    = "/time"
	VERSION = "/version"

	// API密钥管理
	CreateAPIKey = "/auth/api-key"
	GetAPIKeys   = "/auth/api-keys"
	DeleteAPIKey = "/auth/api-key"
	DeriveAPIKey = "/auth/derive-api-key"
	ClosedOnly   = "/auth/ban-status/closed-only"

	// 只读API密钥
	CreateReadonlyAPIKey = "/auth/readonly-api-key"
	GetReadonlyAPIKeys   = "/auth/readonly-api-keys"
	DeleteReadonlyAPIKey = "/auth/readonly-api-key"

	// Builder API密钥
	CreateBuilderAPIKey  = "/auth/builder-api-key"
	GetBuilderAPIKeys    = "/auth/builder-api-key"
	RevokeBuilderAPIKey  = "/auth/builder-api-key"

	// 交易和订单
	Trades              = "/data/trades"
	GetOrderBook        = "/book"
	GetOrderBooks       = "/books"
	GetOrder            = "/data/order/"
	Orders              = "/data/orders"
	PreMigrationOrders  = "/data/pre-migration-orders"
	PostOrder           = "/order"
	PostOrders          = "/orders"
	Cancel              = "/order"
	CancelOrders        = "/orders"
	CancelAll           = "/cancel-all"
	CancelMarketOrders  = "/cancel-market-orders"

	// 价格和市场数据
	MidPoint            = "/midpoint"
	MidPoints           = "/midpoints"
	Price               = "/price"
	GetPrices           = "/prices"
	GetPricesHistory    = "/prices-history"
	GetSpread           = "/spread"
	GetSpreads          = "/spreads"
	GetLastTradePrice   = "/last-trade-price"
	GetLastTradesPrices = "/last-trades-prices"

	// 通知
	GetNotifications  = "/notifications"
	DropNotifications = "/notifications"

	// 余额和授权
	GetBalanceAllowance    = "/balance-allowance"
	UpdateBalanceAllowance = "/balance-allowance/update"

	// 订单评分
	IsOrderScoring   = "/order-scoring"
	AreOrdersScoring = "/orders-scoring"

	// 市场信息
	GetTickSize                  = "/tick-size"
	GetNegRisk                   = "/neg-risk"
	GetFeeRate                   = "/fee-rate"
	GetSamplingSimplifiedMarkets = "/sampling-simplified-markets"
	GetSamplingMarkets           = "/sampling-markets"
	GetSimplifiedMarkets         = "/simplified-markets"
	GetMarkets                   = "/markets"
	GetMarket                    = "/markets/"
	GetMarketByToken             = "/markets-by-token/"
	GetMarketTradesEvents        = "/markets/live-activity/"
	GetClobMarket                = "/clob-markets/"

	// Builder
	GetBuilderTrades   = "/builder/trades"
	GetBuilderFeeRate  = "/fees/builder-fees/"

	// Heartbeat
	PostHeartbeat = "/v1/heartbeats"

	// Rewards
	GetEarningsForUserForDay        = "/rewards/user"
	GetTotalEarningsForUserForDay   = "/rewards/user/total"
	GetLiquidityRewardPercentages   = "/rewards/user/percentages"
	GetRewardsMarketsCurrent        = "/rewards/markets/current"
	GetRewardsMarkets               = "/rewards/markets/"
	GetRewardsEarningsPercentages   = "/rewards/user/markets"

	// RFQ
	CreateRFQRequest      = "/rfq/request"
	CancelRFQRequest      = "/rfq/request"
	GetRFQRequests        = "/rfq/data/requests"
	CreateRFQQuote        = "/rfq/quote"
	CancelRFQQuote        = "/rfq/quote"
	GetRFQRequesterQuotes = "/rfq/data/requester/quotes"
	GetRFQQuoterQuotes    = "/rfq/data/quoter/quotes"
	GetRFQBestQuote       = "/rfq/data/best-quote"
	RFQRequestsAccept     = "/rfq/request/accept"
	RFQQuoteApprove       = "/rfq/quote/approve"
	RFQConfig             = "/rfq/config"
)
