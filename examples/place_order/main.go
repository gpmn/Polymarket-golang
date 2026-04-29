package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/0xNetuser/Polymarket-golang/polymarket"
)

func main() {
	host := os.Getenv("CLOB_HOST")
	if host == "" {
		host = "https://clob.polymarket.com"
	}

	chainIDStr := os.Getenv("CHAIN_ID")
	chainID := 137
	if chainIDStr != "" {
		fmt.Sscanf(chainIDStr, "%d", &chainID)
	}

	privateKey := os.Getenv("PRIVATE_KEY")
	if privateKey == "" {
		log.Fatalf("错误: 必须设置 PRIVATE_KEY 环境变量")
	}

	funder := os.Getenv("FUNDER")

	signatureTypeStr := os.Getenv("SIGNATURE_TYPE")
	signatureType := 0
	if signatureTypeStr != "" {
		fmt.Sscanf(signatureTypeStr, "%d", &signatureType)
	}
	sigTypePtr := &signatureType

	client, err := polymarket.NewClobClient(
		host, chainID, privateKey, nil, sigTypePtr, funder,
	)
	if err != nil {
		log.Fatalf("创建客户端失败: %v", err)
	}

	fmt.Println("=== Polymarket 下单示例 (v2) ===")
	fmt.Printf("地址: %s\n", client.GetAddress())
	fmt.Printf("链ID: %d\n", chainID)
	fmt.Printf("签名类型: %d (0=EOA, 1=PolyProxy, 2=GnosisSafe, 3=Poly1271)\n", signatureType)
	fmt.Println()

	apiKey := os.Getenv("CLOB_API_KEY")
	apiSecret := os.Getenv("CLOB_SECRET")
	apiPassphrase := os.Getenv("CLOB_PASSPHRASE")

	var creds *polymarket.ApiCreds
	if apiKey != "" && apiSecret != "" && apiPassphrase != "" {
		fmt.Println("使用环境变量中的API凭证...")
		creds = &polymarket.ApiCreds{
			APIKey:        apiKey,
			APISecret:     apiSecret,
			APIPassphrase: apiPassphrase,
		}
		client.SetAPICreds(creds)
	} else {
		fmt.Println("未找到API凭证，正在创建或派生...")
		nonce := 0
		creds, err = client.DeriveAPIKey(&nonce)
		if err != nil {
			fmt.Println("派生失败，尝试创建新的API密钥...")
			creds, err = client.CreateAPIKey(&nonce)
			if err != nil {
				log.Fatalf("创建API密钥失败: %v", err)
			}
			fmt.Println("⚠️  新API密钥已创建，请保存以下凭证：")
			fmt.Printf("   API Key: %s\n", creds.APIKey)
			fmt.Printf("   Secret: %s\n", creds.APISecret)
			fmt.Printf("   Passphrase: %s\n", creds.APIPassphrase)
			fmt.Println()
		} else {
			fmt.Println("✓ 成功派生API密钥")
		}
	}

	tokenID := os.Getenv("TOKEN_ID")
	if tokenID == "" {
		log.Fatalf("错误: 必须设置 TOKEN_ID 环境变量")
	}

	orderSide := os.Getenv("ORDER_SIDE")
	if orderSide == "" {
		orderSide = "BUY"
	}
	if orderSide != "BUY" && orderSide != "SELL" {
		log.Fatalf("错误: ORDER_SIDE 必须是 BUY 或 SELL")
	}

	orderType := os.Getenv("ORDER_TYPE")
	useMarketOrder := orderType == "MARKET"

	if useMarketOrder {
		fmt.Println("\n=== 创建市价订单 ===")

		amountStr := os.Getenv("AMOUNT")
		if amountStr == "" {
			log.Fatalf("错误: 市价订单必须设置 AMOUNT 环境变量")
		}
		amount, err := strconv.ParseFloat(amountStr, 64)
		if err != nil {
			log.Fatalf("错误: AMOUNT 必须是有效的数字: %v", err)
		}

		orderTypeStr := os.Getenv("MARKET_ORDER_TYPE")
		if orderTypeStr == "" {
			orderTypeStr = "FOK"
		}
		marketOrderType := polymarket.OrderType(orderTypeStr)

		builderCode := os.Getenv("BUILDER_CODE")

		marketOrderArgs := &polymarket.MarketOrderArgs{
			TokenID:     tokenID,
			Amount:      amount,
			Side:        orderSide,
			Price:       0, // 自动计算
			OrderType:   marketOrderType,
			BuilderCode: builderCode,
		}

		fmt.Printf("订单参数:\n")
		fmt.Printf("  TokenID: %s\n", marketOrderArgs.TokenID)
		fmt.Printf("  Side: %s\n", marketOrderArgs.Side)
		fmt.Printf("  Amount: %.6f\n", marketOrderArgs.Amount)
		fmt.Printf("  OrderType: %s\n", marketOrderType)
		fmt.Println()

		fmt.Println("正在创建并签名订单...")
		signedOrder, err := client.CreateMarketOrder(marketOrderArgs, nil)
		if err != nil {
			log.Fatalf("创建市价订单失败: %v", err)
		}

		fmt.Println("✓ 订单创建成功")
		printOrderV2(signedOrder)

		fmt.Println("\n正在提交订单到交易所...")
		result, err := client.PostOrder(signedOrder, marketOrderType)
		if err != nil {
			log.Fatalf("提交订单失败: %v", err)
		}

		fmt.Println("✓ 订单提交成功")
		printResult(result)

	} else {
		fmt.Println("\n=== 创建限价订单 ===")

		priceStr := os.Getenv("PRICE")
		if priceStr == "" {
			log.Fatalf("错误: 限价订单必须设置 PRICE 环境变量")
		}
		price, err := strconv.ParseFloat(priceStr, 64)
		if err != nil {
			log.Fatalf("错误: PRICE 必须是有效的数字: %v", err)
		}

		sizeStr := os.Getenv("SIZE")
		if sizeStr == "" {
			log.Fatalf("错误: 限价订单必须设置 SIZE 环境变量（条件代币数量）")
		}
		size, err := strconv.ParseFloat(sizeStr, 64)
		if err != nil {
			log.Fatalf("错误: SIZE 必须是有效的数字: %v", err)
		}

		limitOrderType := os.Getenv("LIMIT_ORDER_TYPE")
		if limitOrderType == "" {
			limitOrderType = "GTC"
		}

		expiration := 0
		if limitOrderType == "GTD" {
			expirationStr := os.Getenv("EXPIRATION")
			if expirationStr != "" {
				exp, err := strconv.Atoi(expirationStr)
				if err != nil {
					log.Fatalf("错误: EXPIRATION 必须是有效的Unix时间戳: %v", err)
				}
				expiration = exp
			} else {
				expiration = int(time.Now().Add(30 * 24 * time.Hour).Unix())
			}
		}

		builderCode := os.Getenv("BUILDER_CODE")

		orderArgs := &polymarket.OrderArgs{
			TokenID:     tokenID,
			Price:       price,
			Size:        size,
			Side:        orderSide,
			Expiration:  expiration,
			BuilderCode: builderCode,
		}

		fmt.Printf("订单参数:\n")
		fmt.Printf("  TokenID: %s\n", orderArgs.TokenID)
		fmt.Printf("  Side: %s\n", orderArgs.Side)
		fmt.Printf("  Price: %.6f\n", orderArgs.Price)
		fmt.Printf("  Size: %.6f\n", orderArgs.Size)
		fmt.Printf("  OrderType: %s\n", limitOrderType)
		if orderArgs.Expiration > 0 {
			fmt.Printf("  Expiration: %d (%s)\n", orderArgs.Expiration, time.Unix(int64(orderArgs.Expiration), 0).Format("2006-01-02 15:04:05"))
		} else {
			fmt.Printf("  Expiration: 0 (无过期时间)\n")
		}
		fmt.Println()

		useConvenienceMethod := os.Getenv("USE_CONVENIENCE") != "false"
		if useConvenienceMethod {
			fmt.Println("使用便捷方法 (CreateAndPostOrder)...")
			result, err := client.CreateAndPostOrder(orderArgs, nil)
			if err != nil {
				log.Fatalf("创建并提交订单失败: %v", err)
			}
			fmt.Println("✓ 订单创建并提交成功")
			printResult(result)
		} else {
			fmt.Println("正在创建并签名订单...")
			signedOrder, err := client.CreateOrder(orderArgs, nil)
			if err != nil {
				log.Fatalf("创建订单失败: %v", err)
			}

			fmt.Println("✓ 订单创建成功")
			printOrderV2(signedOrder)

			fmt.Println("\n正在提交订单到交易所...")
			postOrderType := polymarket.OrderType(limitOrderType)
			result, err := client.PostOrder(signedOrder, postOrderType)
			if err != nil {
				log.Fatalf("提交订单失败: %v", err)
			}

			fmt.Println("✓ 订单提交成功")
			printResult(result)
		}
	}

	fmt.Println("\n=== 完成 ===")
}

func printOrderV2(order interface{}) {
	fmt.Println("\n订单详情:")
	if v2Order, ok := order.(*polymarket.SignedOrderV2); ok {
		fmt.Printf("  Salt: %s\n", v2Order.Salt)
		fmt.Printf("  Maker: %s\n", v2Order.Maker)
		fmt.Printf("  Signer: %s\n", v2Order.Signer)
		fmt.Printf("  TokenId: %s\n", v2Order.TokenId)
		fmt.Printf("  MakerAmount: %s\n", v2Order.MakerAmount)
		fmt.Printf("  TakerAmount: %s\n", v2Order.TakerAmount)
		fmt.Printf("  Side: %s\n", v2Order.Side)
		fmt.Printf("  Expiration: %s\n", v2Order.Expiration)
		fmt.Printf("  SignatureType: %d\n", v2Order.SignatureType)
		fmt.Printf("  Timestamp: %s\n", v2Order.Timestamp)
		fmt.Printf("  Builder: %s\n", v2Order.Builder)
		fmt.Printf("  Signature: %s\n", v2Order.Signature)
	} else {
		jsonData, _ := json.MarshalIndent(order, "  ", "  ")
		fmt.Println(string(jsonData))
	}
}

func printResult(result interface{}) {
	fmt.Println("\nAPI响应:")
	jsonData, err := json.MarshalIndent(result, "  ", "  ")
	if err != nil {
		fmt.Printf("  原始数据: %+v\n", result)
	} else {
		fmt.Println(string(jsonData))
	}
}
