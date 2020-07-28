package types

import (
	"encoding/json"
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"sort"
	"strconv"
)

const (
	PowerFunction     = "power_function"
	SigmoidFunction   = "sigmoid_function"
	SwapperFunction   = "swapper_function"
	AugmentedFunction = "augmented_function"
	DoNotModifyField  = "[do-not-modify]"

	AnyNumberOfReserveTokens = -1
)

type FunctionParamRestrictions func(paramsMap map[string]sdk.Dec) sdk.Error

var (
	RequiredParamsForFunctionType = map[string][]string{
		PowerFunction:     {"m", "n", "c"},
		SigmoidFunction:   {"a", "b", "c"},
		SwapperFunction:   nil,
		AugmentedFunction: {"d0", "p0", "theta", "kappa"},
	}

	NoOfReserveTokensForFunctionType = map[string]int{
		PowerFunction:     AnyNumberOfReserveTokens,
		SigmoidFunction:   AnyNumberOfReserveTokens,
		SwapperFunction:   2,
		AugmentedFunction: AnyNumberOfReserveTokens,
	}

	ExtraParameterRestrictions = map[string]FunctionParamRestrictions{
		PowerFunction:     powerParameterRestrictions,
		SigmoidFunction:   sigmoidParameterRestrictions,
		SwapperFunction:   nil,
		AugmentedFunction: nil, // TODO?
	}
)

type FunctionParam struct {
	Param string  `json:"param" yaml:"param"`
	Value sdk.Dec `json:"value" yaml:"value"`
}

func NewFunctionParam(param string, value sdk.Dec) FunctionParam {
	return FunctionParam{
		Param: param,
		Value: value,
	}
}

type FunctionParams []FunctionParam

func (fps FunctionParams) Validate(functionType string) sdk.Error {
	// Come up with list of expected parameters
	expectedParams, err := GetRequiredParamsForFunctionType(functionType)
	if err != nil {
		return err
	}

	// Check that number of params is as expected
	if len(fps) != len(expectedParams) {
		return ErrIncorrectNumberOfFunctionParameters(DefaultCodespace, len(expectedParams))
	}

	// Check that params match and all values are non-negative
	paramsMap := fps.AsMap()
	for _, p := range expectedParams {
		val, ok := paramsMap[p]
		if !ok {
			return ErrFunctionParameterMissingOrNonFloat(DefaultCodespace, p)
		} else if val.IsNegative() {
			return ErrArgumentCannotBeNegative(DefaultCodespace, "FunctionParams:"+p)
		}
	}

	// Get extra function parameter restrictions
	extraRestrictions, err := GetExceptionsForFunctionType(functionType)
	if err != nil {
		return err
	}
	if extraRestrictions != nil {
		err := extraRestrictions(paramsMap)
		if err != nil {
			return err
		}
	}

	return nil
}

func (fps FunctionParams) String() (result string) {
	output, err := json.Marshal(fps)
	if err != nil {
		panic(err)
	}
	return string(output)
}

func (fps FunctionParams) AsMap() (paramsMap map[string]sdk.Dec) {
	paramsMap = make(map[string]sdk.Dec)
	for _, fp := range fps {
		paramsMap[fp.Param] = fp.Value
	}
	return paramsMap
}

func powerParameterRestrictions(paramsMap map[string]sdk.Dec) sdk.Error {
	// Power exception 1: n must be an integer, otherwise x^n loop does not work
	val, ok := paramsMap["n"]
	if !ok {
		panic("did not find parameter n for power function")
	} else if !val.TruncateDec().Equal(val) {
		return ErrArgumentMustBeInteger(DefaultCodespace, "FunctionParams:n")
	}
	return nil
}

func sigmoidParameterRestrictions(paramsMap map[string]sdk.Dec) sdk.Error {
	// Sigmoid exception 1: c != 0, otherwise we run into divisions by zero
	val, ok := paramsMap["c"]
	if !ok {
		panic("did not find parameter c for sigmoid function")
	} else if !val.IsPositive() {
		return ErrArgumentMustBePositive(DefaultCodespace, "FunctionParams:c")
	}
	return nil
}

type Bond struct {
	Token                  string           `json:"token" yaml:"token"`
	Name                   string           `json:"name" yaml:"name"`
	Description            string           `json:"description" yaml:"description"`
	Creator                sdk.AccAddress   `json:"creator" yaml:"creator"`
	FunctionType           string           `json:"function_type" yaml:"function_type"`
	FunctionParameters     FunctionParams   `json:"function_parameters" yaml:"function_parameters"`
	ReserveTokens          []string         `json:"reserve_tokens" yaml:"reserve_tokens"`
	ReserveAddress         sdk.AccAddress   `json:"reserve_address" yaml:"reserve_address"`
	TxFeePercentage        sdk.Dec          `json:"tx_fee_percentage" yaml:"tx_fee_percentage"`
	ExitFeePercentage      sdk.Dec          `json:"exit_fee_percentage" yaml:"exit_fee_percentage"`
	FeeAddress             sdk.AccAddress   `json:"fee_address" yaml:"fee_address"`
	MaxSupply              sdk.Coin         `json:"max_supply" yaml:"max_supply"`
	OrderQuantityLimits    sdk.Coins        `json:"order_quantity_limits" yaml:"order_quantity_limits"`
	SanityRate             sdk.Dec          `json:"sanity_rate" yaml:"sanity_rate"`
	SanityMarginPercentage sdk.Dec          `json:"sanity_margin_percentage" yaml:"sanity_margin_percentage"`
	CurrentSupply          sdk.Coin         `json:"current_supply" yaml:"current_supply"`
	AllowSells             string           `json:"allow_sells" yaml:"allow_sells"`
	Signers                []sdk.AccAddress `json:"signers" yaml:"signers"`
	BatchBlocks            sdk.Uint         `json:"batch_blocks" yaml:"batch_blocks"`
}

func NewBond(token, name, description string, creator sdk.AccAddress,
	functionType string, functionParameters FunctionParams,
	reserveTokens []string, reserveAdddress sdk.AccAddress,
	txFeePercentage, exitFeePercentage sdk.Dec, feeAddress sdk.AccAddress,
	maxSupply sdk.Coin, orderQuantityLimits sdk.Coins, sanityRate,
	sanityMarginPercentage sdk.Dec, allowSells string, signers []sdk.AccAddress,
	batchBlocks sdk.Uint) Bond {

	// Ensure tokens and coins are sorted
	sort.Strings(reserveTokens)
	orderQuantityLimits = orderQuantityLimits.Sort()

	return Bond{
		Token:                  token,
		Name:                   name,
		Description:            description,
		Creator:                creator,
		FunctionType:           functionType,
		FunctionParameters:     functionParameters,
		ReserveTokens:          reserveTokens,
		ReserveAddress:         reserveAdddress,
		TxFeePercentage:        txFeePercentage,
		ExitFeePercentage:      exitFeePercentage,
		FeeAddress:             feeAddress,
		MaxSupply:              maxSupply,
		OrderQuantityLimits:    orderQuantityLimits,
		SanityRate:             sanityRate,
		SanityMarginPercentage: sanityMarginPercentage,
		CurrentSupply:          sdk.NewCoin(token, sdk.ZeroInt()),
		AllowSells:             allowSells,
		Signers:                signers,
		BatchBlocks:            batchBlocks,
	}
}

//noinspection GoNilness
func (bond Bond) GetNewReserveDecCoins(amount sdk.Dec) (coins sdk.DecCoins) {
	for _, r := range bond.ReserveTokens {
		coins = coins.Add(sdk.DecCoins{sdk.NewDecCoinFromDec(r, amount)})
	}
	return coins
}

func (bond Bond) GetPricesAtSupply(supply sdk.Int) (result sdk.DecCoins, err sdk.Error) {
	if supply.IsNegative() {
		panic(fmt.Sprintf("negative supply for bond %s", bond))
	}

	args := bond.FunctionParameters.AsMap()
	x := sdk.NewDecFromInt(supply)
	switch bond.FunctionType {
	case PowerFunction:
		// TODO: should try using simpler approach, especially
		//       if function params are changed to decimals
		m := args["m"]
		n64 := args["n"].TruncateInt64() // enforced by powerParameterRestrictions
		c := args["c"]
		result = bond.GetNewReserveDecCoins(
			Power(x, uint64(n64)).Mul(m).Add(c))
	case SigmoidFunction:
		a := args["a"]
		b := args["b"]
		c := args["c"]
		temp1 := x.Sub(b)
		temp2 := temp1.Mul(temp1).Add(c)
		temp3 := temp2.ApproxSqrt()
		result = bond.GetNewReserveDecCoins(a.Mul(temp1.Quo(temp3).Add(sdk.OneDec())))
	case AugmentedFunction: // TODO: fix?
		kappaFloat64, err := strconv.ParseFloat(args["kappa"].String(), 64)
		if err != nil {
			panic(err)
		}
		V0Float64, err := strconv.ParseFloat(args["V0"].String(), 64)
		if err != nil {
			panic(err)
		}
		resFloat64 := Reserve(float64(supply.Int64()), kappaFloat64, V0Float64)
		spotPriceFloat64 := SpotPrice(resFloat64, kappaFloat64, V0Float64)
		spotPriceDec := sdk.MustNewDecFromStr(strconv.FormatFloat(spotPriceFloat64, 'f', 6, 64))
		result = bond.GetNewReserveDecCoins(spotPriceDec)
	case SwapperFunction:
		return nil, ErrFunctionNotAvailableForFunctionType(DefaultCodespace)
	default:
		panic("unrecognized function type")
	}

	if result.IsAnyNegative() {
		// assumes that the curve is above the x-axis and does not intersect it
		panic(fmt.Sprintf("negative price result for bond %s", bond))
	}
	return result, nil
}

func (bond Bond) GetCurrentPricesPT(reserveBalances sdk.Coins) (sdk.DecCoins, sdk.Error) {
	// Note: PT stands for "per token"
	switch bond.FunctionType {
	case PowerFunction:
		fallthrough
	case SigmoidFunction:
		fallthrough
	case AugmentedFunction:
		return bond.GetPricesAtSupply(bond.CurrentSupply.Amount)
	case SwapperFunction:
		return bond.GetPricesToMint(sdk.OneInt(), reserveBalances)
	default:
		panic("unrecognized function type")
	}
}

func (bond Bond) CurveIntegral(supply sdk.Int) (result sdk.Dec) {
	if supply.IsNegative() {
		panic(fmt.Sprintf("negative supply for bond %s", bond))
	}

	args := bond.FunctionParameters.AsMap()
	x := sdk.NewDecFromInt(supply)
	switch bond.FunctionType {
	case PowerFunction:
		// TODO: should try using simpler approach, especially
		//       if function params are changed to decimals
		m := args["m"]
		n, n64 := args["n"], args["n"].TruncateInt64() // enforced by powerParameterRestrictions
		c := args["c"]
		temp1 := Power(x, uint64(n64+1))
		temp2 := temp1.Mul(m).Quo(n.Add(sdk.OneDec()))
		temp3 := x.Mul(c)
		result = temp2.Add(temp3)
	case SigmoidFunction:
		a := args["a"]
		b := args["b"]
		c := args["c"]
		temp1 := x.Sub(b)
		temp2 := temp1.Mul(temp1).Add(c)
		temp3 := temp2.ApproxSqrt()
		temp5 := a.Mul(temp3.Add(x))
		constant := a.Mul((b.Mul(b).Add(c)).ApproxSqrt())
		result = temp5.Sub(constant)
	case AugmentedFunction: // TODO: fix?
		kappaFloat64, err := strconv.ParseFloat(args["kappa"].String(), 64)
		if err != nil {
			panic(err)
		}
		V0Float64, err := strconv.ParseFloat(args["V0"].String(), 64)
		if err != nil {
			panic(err)
		}
		resFloat64 := Reserve(float64(supply.Int64()), kappaFloat64, V0Float64)
		result = sdk.MustNewDecFromStr(strconv.FormatFloat(resFloat64, 'f', 6, 64))
	case SwapperFunction:
		panic("invalid function for function type")
	default:
		panic("unrecognized function type")
	}

	if result.IsNegative() {
		// assumes that the curve is above the x-axis and does not intersect it
		panic(fmt.Sprintf("negative integral result for bond %s", bond))
	}
	return result
}

func (bond Bond) GetReserveDeltaForLiquidityDelta(mintOrBurn sdk.Int, reserveBalances sdk.Coins) sdk.DecCoins {
	if mintOrBurn.IsNegative() {
		panic(fmt.Sprintf("negative liquidity delta for bond %s", bond))
	} else if reserveBalances.IsAnyNegative() {
		panic(fmt.Sprintf("negative reserve balance for bond %s", bond))
	}

	switch bond.FunctionType {
	case PowerFunction:
		fallthrough
	case SigmoidFunction:
		fallthrough
	case AugmentedFunction:
		panic("invalid function for function type")
	case SwapperFunction:
		resToken1 := bond.ReserveTokens[0]
		resToken2 := bond.ReserveTokens[1]
		resBalance1 := sdk.NewDecFromInt(reserveBalances.AmountOf(resToken1))
		resBalance2 := sdk.NewDecFromInt(reserveBalances.AmountOf(resToken2))
		mintOrBurnDec := sdk.NewDecFromInt(mintOrBurn)

		// Using Uniswap formulae: x' = (1+-α)x = x +- Δx, where α = Δx/x
		// Where x is any of the two reserve balances or the current supply
		// and x' is any of the updated reserve balances or the updated supply
		// By making Δx subject of the formula: Δx = αx
		alpha := mintOrBurnDec.Quo(sdk.NewDecFromInt(bond.CurrentSupply.Amount))

		result := sdk.DecCoins{
			sdk.NewDecCoinFromDec(resToken1, alpha.Mul(resBalance1)),
			sdk.NewDecCoinFromDec(resToken2, alpha.Mul(resBalance2)),
		}
		if result.IsAnyNegative() {
			panic(fmt.Sprintf("negative reserve delta result for bond %s", bond))
		}
		return result
	default:
		panic("unrecognized function type")
	}
}

func (bond Bond) GetPricesToMint(mint sdk.Int, reserveBalances sdk.Coins) (sdk.DecCoins, sdk.Error) {
	if mint.IsNegative() {
		panic(fmt.Sprintf("negative mint amount for bond %s", bond))
	} else if reserveBalances.IsAnyNegative() {
		panic(fmt.Sprintf("negative reserve balance for bond %s", bond))
	}

	switch bond.FunctionType {
	case PowerFunction:
		fallthrough
	case SigmoidFunction:
		var priceToMint sdk.Dec
		result := bond.CurveIntegral(bond.CurrentSupply.Amount.Add(mint))
		if reserveBalances.Empty() {
			priceToMint = result
		} else {
			// Reserve balances should all be equal given that we are always
			// applying the same additions/subtractions to all reserve balances
			commonReserveBalance := sdk.NewDecFromInt(reserveBalances[0].Amount)
			priceToMint = result.Sub(commonReserveBalance)
		}
		if priceToMint.IsNegative() {
			// Negative priceToMint means that the previous buyer overpaid
			// to the point that the price for this buyer is covered. However,
			// we still charge this buyer at least one token.
			priceToMint = sdk.OneDec()
		}
		return bond.GetNewReserveDecCoins(priceToMint), nil
	case AugmentedFunction: // TODO: fix?
		var reserveBalance sdk.Dec
		if reserveBalances.Empty() {
			reserveBalance = sdk.ZeroDec()
		} else {
			// Reserve balances should all be equal given that we are always
			// applying the same additions/subtractions to all reserve balances.
			// Thus we can pick the first reserve balance as the global balance.
			reserveBalance = sdk.NewDecFromInt(reserveBalances[0].Amount)
		}

		args := bond.FunctionParameters.AsMap()

		kappaFloat64, err := strconv.ParseFloat(args["kappa"].String(), 64)
		if err != nil {
			panic(err)
		}
		V0Float64, err := strconv.ParseFloat(args["V0"].String(), 64)
		if err != nil {
			panic(err)
		}
		resFloat64, err := strconv.ParseFloat(reserveBalance.String(), 64)
		if err != nil {
			panic(err)
		}
		supplyFloat64, err := strconv.ParseFloat(bond.CurrentSupply.Amount.String(), 64)
		if err != nil {
			panic(err)
		}
		deltaSFloat64, err := strconv.ParseFloat(mint.String(), 64)
		if err != nil {
			panic(err)
		}

		returnForBurnFloat64, _ := MintAlt(deltaSFloat64, resFloat64, supplyFloat64, kappaFloat64, V0Float64)
		returnForBurnDec := sdk.MustNewDecFromStr(strconv.FormatFloat(returnForBurnFloat64, 'f', 6, 64))
		return bond.GetNewReserveDecCoins(returnForBurnDec), nil
	case SwapperFunction:
		if bond.CurrentSupply.Amount.IsZero() {
			return nil, ErrFunctionRequiresNonZeroCurrentSupply(DefaultCodespace)
		}
		return bond.GetReserveDeltaForLiquidityDelta(mint, reserveBalances), nil
	default:
		panic("unrecognized function type")
	}
	// Note: fees have to be added to these prices to get actual prices
}

func (bond Bond) GetReturnsForBurn(burn sdk.Int, reserveBalances sdk.Coins) sdk.DecCoins {
	if burn.IsNegative() {
		panic(fmt.Sprintf("negative burn amount for bond %s", bond))
	} else if reserveBalances.IsAnyNegative() {
		panic(fmt.Sprintf("negative reserve balance for bond %s", bond))
	}

	switch bond.FunctionType {
	case PowerFunction:
		fallthrough
	case SigmoidFunction:
		result := bond.CurveIntegral(bond.CurrentSupply.Amount.Sub(burn))

		var reserveBalance sdk.Dec
		if reserveBalances.Empty() {
			reserveBalance = sdk.ZeroDec()
		} else {
			// Reserve balances should all be equal given that we are always
			// applying the same additions/subtractions to all reserve balances.
			// Thus we can pick the first reserve balance as the global balance.
			reserveBalance = sdk.NewDecFromInt(reserveBalances[0].Amount)
		}

		if result.GT(reserveBalance) {
			panic("not enough reserve available for burn")
		} else {
			returnForBurn := reserveBalance.Sub(result)
			return bond.GetNewReserveDecCoins(returnForBurn)
			// TODO: investigate possibility of negative returnForBurn
		}
	case AugmentedFunction: // TODO: fix?

		var reserveBalance sdk.Dec
		if reserveBalances.Empty() {
			reserveBalance = sdk.ZeroDec()
		} else {
			// Reserve balances should all be equal given that we are always
			// applying the same additions/subtractions to all reserve balances.
			// Thus we can pick the first reserve balance as the global balance.
			reserveBalance = sdk.NewDecFromInt(reserveBalances[0].Amount)
		}

		args := bond.FunctionParameters.AsMap()

		kappaFloat64, err := strconv.ParseFloat(args["kappa"].String(), 64)
		if err != nil {
			panic(err)
		}
		V0Float64, err := strconv.ParseFloat(args["V0"].String(), 64)
		if err != nil {
			panic(err)
		}
		resFloat64, err := strconv.ParseFloat(reserveBalance.String(), 64)
		if err != nil {
			panic(err)
		}
		supplyFloat64, err := strconv.ParseFloat(bond.CurrentSupply.Amount.String(), 64)
		if err != nil {
			panic(err)
		}
		deltaSFloat64, err := strconv.ParseFloat(burn.String(), 64)
		if err != nil {
			panic(err)
		}

		returnForBurnFloat64, _ := Withdraw(deltaSFloat64, resFloat64, supplyFloat64, kappaFloat64, V0Float64)
		returnForBurnDec := sdk.MustNewDecFromStr(strconv.FormatFloat(returnForBurnFloat64, 'f', 6, 64))
		return bond.GetNewReserveDecCoins(returnForBurnDec)
	case SwapperFunction:
		return bond.GetReserveDeltaForLiquidityDelta(burn, reserveBalances)
	default:
		panic("unrecognized function type")
	}
	// Note: fees have to be deducted from these returns to get actual returns
}

func (bond Bond) GetReturnsForSwap(from sdk.Coin, toToken string, reserveBalances sdk.Coins) (returns sdk.Coins, txFee sdk.Coin, err sdk.Error) {
	if from.IsNegative() {
		panic(fmt.Sprintf("negative from amount for bond %s", bond))
	} else if reserveBalances.IsAnyNegative() {
		panic(fmt.Sprintf("negative reserve balance for bond %s", bond))
	}

	switch bond.FunctionType {
	case PowerFunction:
		fallthrough
	case SigmoidFunction:
		fallthrough
	case AugmentedFunction:
		return nil, sdk.Coin{}, ErrFunctionNotAvailableForFunctionType(DefaultCodespace)
	case SwapperFunction:
		// Check that from and to are reserve tokens
		if from.Denom != bond.ReserveTokens[0] && from.Denom != bond.ReserveTokens[1] {
			return nil, sdk.Coin{}, ErrTokenIsNotAValidReserveToken(DefaultCodespace, from.Denom)
		} else if toToken != bond.ReserveTokens[0] && toToken != bond.ReserveTokens[1] {
			return nil, sdk.Coin{}, ErrTokenIsNotAValidReserveToken(DefaultCodespace, toToken)
		}

		inAmt := from.Amount
		inRes := reserveBalances.AmountOf(from.Denom)
		outRes := reserveBalances.AmountOf(toToken)

		// Calculate fee to get the adjusted input amount
		txFee = bond.GetTxFee(sdk.NewDecCoinFromCoin(from))
		inAmt = inAmt.Sub(txFee.Amount) // adjusted input

		// Check that at least 1 token is going in
		if inAmt.IsZero() {
			return nil, sdk.Coin{}, ErrSwapAmountTooSmallToGiveAnyReturn(DefaultCodespace, from.Denom, toToken)
		}

		// Calculate output amount using Uniswap formula: Δy = (Δx*y)/(x+Δx)
		outAmt := inAmt.Mul(outRes).Quo(inRes.Add(inAmt))

		// Check that not giving out all of the available outRes or nothing at all
		if outAmt.Equal(outRes) {
			return nil, sdk.Coin{}, ErrSwapAmountCausesReserveDepletion(DefaultCodespace, from.Denom, toToken)
		} else if outAmt.IsZero() {
			return nil, sdk.Coin{}, ErrSwapAmountTooSmallToGiveAnyReturn(DefaultCodespace, from.Denom, toToken)
		} else if outAmt.IsNegative() {
			panic(fmt.Sprintf("negative return for swap result for bond %s", bond))
		}

		return sdk.Coins{sdk.NewCoin(toToken, outAmt)}, txFee, nil
	default:
		panic("unrecognized function type")
	}
}

func (bond Bond) GetTxFee(reserveAmount sdk.DecCoin) sdk.Coin {
	feeAmount := bond.TxFeePercentage.QuoInt64(100).Mul(reserveAmount.Amount)
	return RoundFee(sdk.NewDecCoinFromDec(reserveAmount.Denom, feeAmount))
}

func (bond Bond) GetExitFee(reserveAmount sdk.DecCoin) sdk.Coin {
	feeAmount := bond.ExitFeePercentage.QuoInt64(100).Mul(reserveAmount.Amount)
	return RoundFee(sdk.NewDecCoinFromDec(reserveAmount.Denom, feeAmount))
}

//noinspection GoNilness
func (bond Bond) GetTxFees(reserveAmounts sdk.DecCoins) (fees sdk.Coins) {
	for _, r := range reserveAmounts {
		fees = fees.Add(sdk.Coins{bond.GetTxFee(r)})
	}
	return fees
}

//noinspection GoNilness
func (bond Bond) GetExitFees(reserveAmounts sdk.DecCoins) (fees sdk.Coins) {
	for _, r := range reserveAmounts {
		fees = fees.Add(sdk.Coins{bond.GetExitFee(r)})
	}
	return fees
}

func (bond Bond) SignersEqualTo(signers []sdk.AccAddress) bool {
	if len(bond.Signers) != len(signers) {
		return false
	}

	// Note: this also enforces ORDER of signatures to be the same
	for i := range signers {
		if !bond.Signers[i].Equals(signers[i]) {
			return false
		}
	}

	return true
}

func (bond Bond) ReserveDenomsEqualTo(coins sdk.Coins) bool {
	if len(bond.ReserveTokens) != len(coins) {
		return false
	}

	for _, d := range bond.ReserveTokens {
		if coins.AmountOf(d).IsZero() {
			return false
		}
	}

	return true
}

func (bond Bond) AnyOrderQuantityLimitsExceeded(amounts sdk.Coins) bool {
	return amounts.IsAnyGT(bond.OrderQuantityLimits)
}

func (bond Bond) ReservesViolateSanityRate(newReserves sdk.Coins) bool {

	if bond.SanityRate.IsZero() {
		return false
	}

	// Get new rate from new balances
	resToken1 := bond.ReserveTokens[0]
	resToken2 := bond.ReserveTokens[1]
	resBalance1 := sdk.NewDecFromInt(newReserves.AmountOf(resToken1))
	resBalance2 := sdk.NewDecFromInt(newReserves.AmountOf(resToken2))
	exchangeRate := resBalance1.Quo(resBalance2)

	// Get max and min acceptable rates
	sanityMarginDecimal := bond.SanityMarginPercentage.Quo(sdk.NewDec(100))
	upperPercentage := sdk.OneDec().Add(sanityMarginDecimal)
	lowerPercentage := sdk.OneDec().Sub(sanityMarginDecimal)
	maxRate := bond.SanityRate.Mul(upperPercentage)
	minRate := bond.SanityRate.Mul(lowerPercentage)

	// If min rate is negative, change to zero
	if minRate.IsNegative() {
		minRate = sdk.ZeroDec()
	}

	return exchangeRate.LT(minRate) || exchangeRate.GT(maxRate)
}
