package polymarket

import (
	"fmt"
	"math"
)

const (
	minFeeSlippagePercentage = 1.0
	maxFeeSlippagePercentage = 100.0
)

// validateFeeSlippage validates the fee_slippage value.
// fee_slippage must be 0 or a percentage between 1 and 100.
func validateFeeSlippage(feeSlippage float64) error {
	if math.IsNaN(feeSlippage) || math.IsInf(feeSlippage, 0) {
		return fmt.Errorf("fee_slippage must be 0 or a percentage between 1 and 100")
	}
	if feeSlippage < 0 || feeSlippage > maxFeeSlippagePercentage || (feeSlippage > 0 && feeSlippage < minFeeSlippagePercentage) {
		return fmt.Errorf("fee_slippage must be 0 or a percentage between 1 and 100")
	}
	return nil
}

// AdjustBuyAmountForFees calculates the fee-adjusted buy amount.
// If the user's USDC balance is insufficient to cover amount + fees,
// the order amount is reduced so that the user can afford the fees.
func AdjustBuyAmountForFees(
	amount float64,
	price float64,
	userUSDCBalance float64,
	feeRate float64,
	feeExponent float64,
	builderTakerFeeRate float64,
	feeSlippage float64,
) float64 {
	if err := validateFeeSlippage(feeSlippage); err != nil {
		// Invalid slippage defaults to 0 (no slippage buffer)
		feeSlippage = 0
	}

	platformFeeRate := feeRate * math.Pow(price*(1-price), feeExponent)
	effectivePlatformFeeRate := platformFeeRate * (1 + feeSlippage/100)
	feeBaseAmount := math.Min(amount, userUSDCBalance)
	platformFee := (feeBaseAmount / price) * effectivePlatformFeeRate
	builderFee := feeBaseAmount * builderTakerFeeRate
	totalCost := amount + platformFee + builderFee

	if userUSDCBalance <= totalCost {
		adjusted := userUSDCBalance - platformFee - builderFee
		if adjusted < 0 {
			return 0
		}
		return adjusted
	}
	return amount
}
