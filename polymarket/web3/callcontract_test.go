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

// TestCallContractBlockOverrides 验证 callContract 发送正确的 eth_call 参数：
// 1. 不包含任何 gas 字段（gas, gasPrice, maxFeePerGas, maxPriorityFeePerGas）
// 2. 包含 blockOverrides 第 4 参数，其中 baseFeePerGas = "0x0"
// 3. 正确返回合约调用结果
func TestCallContractBlockOverrides(t *testing.T) {
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

	// 验证参数数量：args, blockTag, stateOverrides(null), blockOverrides
	if len(req.Params) != 4 {
		t.Fatalf("expected 4 params, got %d", len(req.Params))
	}

	// 验证第 1 个参数（交易参数）：只包含 to 和 data，不含 gas 字段
	var args map[string]interface{}
	if err := json.Unmarshal(req.Params[0], &args); err != nil {
		t.Fatalf("failed to parse call args: %v", err)
	}

	if _, ok := args["to"]; !ok {
		t.Error("call args missing 'to' field")
	}
	if _, ok := args["data"]; !ok {
		t.Error("call args missing 'data' field")
	}

	// 确保没有 gas 相关字段
	gasFields := []string{"gas", "gasPrice", "maxFeePerGas", "maxPriorityFeePerGas", "gasFeeCap", "gasTipCap"}
	for _, field := range gasFields {
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

	// 验证第 3 个参数：state overrides = null
	if string(req.Params[2]) != "null" {
		t.Errorf("expected state overrides to be null, got %s", string(req.Params[2]))
	}

	// 验证第 4 个参数：block overrides 包含 baseFeePerGas = "0x0"
	var blockOverrides map[string]interface{}
	if err := json.Unmarshal(req.Params[3], &blockOverrides); err != nil {
		t.Fatalf("failed to parse block overrides: %v", err)
	}

	baseFee, ok := blockOverrides["baseFeePerGas"]
	if !ok {
		t.Fatal("block overrides missing 'baseFeePerGas' field")
	}
	if baseFee != "0x0" {
		t.Errorf("expected baseFeePerGas '0x0', got '%v'", baseFee)
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
