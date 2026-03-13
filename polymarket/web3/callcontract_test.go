package web3

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

// TestCallContractGasPrice 验证 callContract 发送正确的 eth_call 参数：
// 1. 包含 gasPrice（高值，用于通过 baseFee 校验）
// 2. 不包含 from 字段（避免 Bor 节点注入冲突的 gasPrice）
// 3. 不包含 EIP-1559 字段（maxFeePerGas, maxPriorityFeePerGas）
// 4. 只发送 2 个参数（args + blockTag），不发送 blockOverrides
// 5. 正确返回合约调用结果
func TestCallContractGasPrice(t *testing.T) {
	var capturedBody []byte

	// 模拟 RPC 服务器，返回一个固定的 32 字节结果
	mockResult := "0x000000000000000000000000abcdefabcdefabcdefabcdefabcdefabcdefabcd"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":"` + mockResult + `"}`))
	}))
	defer ts.Close()

	rpcClient, err := rpc.Dial(ts.URL)
	if err != nil {
		t.Fatalf("failed to dial RPC: %v", err)
	}
	defer rpcClient.Close()

	base := &BaseWeb3Client{
		client:    ethclient.NewClient(rpcClient),
		rpcClient: rpcClient,
	}

	to := common.HexToAddress("0x4bFb41d5B3570DeFd03C39a9A4D8dE6Bd8B8982E")
	data := common.FromHex("0x06fdde03") // name() selector

	result, err := base.callContract(context.Background(), &to, data)
	if err != nil {
		t.Fatalf("callContract failed: %v", err)
	}

	// 验证返回结果非空
	if len(result) == 0 {
		t.Fatal("expected non-empty result")
	}

	// 解析捕获的 JSON-RPC 请求
	var req struct {
		Method string            `json:"method"`
		Params []json.RawMessage `json:"params"`
	}
	if err := json.Unmarshal(capturedBody, &req); err != nil {
		t.Fatalf("failed to parse captured request: %v", err)
	}

	// 验证方法名
	if req.Method != "eth_call" {
		t.Errorf("expected method eth_call, got %s", req.Method)
	}

	// 验证参数数量：只有 args + blockTag（2 个参数）
	if len(req.Params) != 2 {
		t.Fatalf("expected 2 params, got %d", len(req.Params))
	}

	// 验证第 1 个参数（交易参数）
	var args map[string]interface{}
	if err := json.Unmarshal(req.Params[0], &args); err != nil {
		t.Fatalf("failed to parse call args: %v", err)
	}

	// 必须包含 to、data、gasPrice
	if _, ok := args["to"]; !ok {
		t.Error("call args missing 'to' field")
	}
	if _, ok := args["data"]; !ok {
		t.Error("call args missing 'data' field")
	}
	if _, ok := args["gasPrice"]; !ok {
		t.Error("call args missing 'gasPrice' field")
	}

	// gasPrice 应该是 1000 Gwei = 1e15 = "0x38d7ea4c68000"
	gasPrice, ok := args["gasPrice"].(string)
	if !ok {
		t.Fatal("gasPrice should be a hex string")
	}
	if gasPrice != "0x38d7ea4c68000" {
		t.Errorf("expected gasPrice '0x38d7ea4c68000' (1000 Gwei), got '%s'", gasPrice)
	}

	// 不能包含 from 字段（避免 Bor 注入 gasPrice 冲突）
	if _, ok := args["from"]; ok {
		t.Error("call args should NOT contain 'from' field")
	}

	// 不能包含 EIP-1559 字段
	eip1559Fields := []string{"maxFeePerGas", "maxPriorityFeePerGas"}
	for _, field := range eip1559Fields {
		if _, ok := args[field]; ok {
			t.Errorf("call args should NOT contain '%s' field, but it does", field)
		}
	}

	// 验证第 2 个参数：block tag = "latest"
	var blockTag string
	if err := json.Unmarshal(req.Params[1], &blockTag); err != nil {
		t.Fatalf("failed to parse block tag: %v", err)
	}
	if blockTag != "latest" {
		t.Errorf("expected block tag 'latest', got '%s'", blockTag)
	}
}

// TestCallContractError 验证 callContract 正确传播 RPC 错误
func TestCallContractError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"jsonrpc":"2.0","id":1,"error":{"code":-32000,"message":"execution reverted"}}`))
	}))
	defer ts.Close()

	rpcClient, err := rpc.Dial(ts.URL)
	if err != nil {
		t.Fatalf("failed to dial RPC: %v", err)
	}
	defer rpcClient.Close()

	base := &BaseWeb3Client{
		client:    ethclient.NewClient(rpcClient),
		rpcClient: rpcClient,
	}

	to := common.HexToAddress("0x4bFb41d5B3570DeFd03C39a9A4D8dE6Bd8B8982E")
	_, err = base.callContract(context.Background(), &to, []byte{0x01})
	if err == nil {
		t.Fatal("expected error from RPC, got nil")
	}
}
