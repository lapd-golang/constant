package params

import (
	"github.com/ninjadotorg/constant/common"
)

type Oracle struct {
	// TODO(@0xankylosaurus): generic prices (ETH, BTC, ...) instead of just bonds
	Bonds    map[string]uint64 // key: bondTypeID, value: price
	DCBToken uint64            // against USD
	GOVToken uint64            // against USD
	Constant uint64            // against USD
	ETH      uint64            // against USD
	BTC      uint64            // against USD
}

type LoanParams struct {
	InterestRate     uint64 `json:"InterestRate"`     // basis points, e.g. 125 represents 1.25%
	Maturity         uint64 `json:"Maturity"`         // in number of blocks
	LiquidationStart uint64 `json:"LiquidationStart"` // ratio between collateral and debt to start auto-liquidation, stored in basis points
}

func NewLoanParams(interestRate uint64, maturity uint64, liquidationStart uint64) *LoanParams {
	return &LoanParams{InterestRate: interestRate, Maturity: maturity, LiquidationStart: liquidationStart}
}

func NewLoanParamsFromJson(data interface{}) *LoanParams {
	loanParamsData := data.(map[string]interface{})
	loanParams := NewLoanParams(
		uint64(loanParamsData["InterestRate"].(float64)),
		uint64(loanParamsData["Maturity"].(float64)),
		uint64(loanParamsData["LiquidationStart"].(float64)),
	)
	return loanParams
}

func NewListLoanParamsFromJson(data interface{}) []LoanParams {
	listLoanParamsData := common.InterfaceSlice(data)
	if listLoanParamsData == nil {
		panic("listLoanParamsData")
	}
	listLoanParams := make([]LoanParams, 0)

	for _, loanParamsData := range listLoanParamsData {
		listLoanParams = append(listLoanParams, *NewLoanParamsFromJson(loanParamsData))
	}
	return listLoanParams
}

type DCBParams struct {
	ListSaleData             []SaleData
	MinLoanResponseRequire   uint8
	MinCMBApprovalRequire    uint8
	LateWithdrawResponseFine uint64 // CST penalty for each CMB's late withdraw response
	SaleDCBTokensByUSDData   *SaleDCBTokensByUSDData

	// TODO(@0xbunyip): read loan params from proposal instead of storing and reading separately
	ListLoanParams []LoanParams // params for collateralized loans of Constant
}

func NewDCBParams(
	listSaleData []SaleData,
	minLoanResponseRequire uint8,
	minCMBApprovalRequire uint8,
	lateWithdrawResponseFine uint64,
	saleDCBTokensByUSDData *SaleDCBTokensByUSDData,
	listLoanParams []LoanParams,
) *DCBParams {
	return &DCBParams{
		ListSaleData:             listSaleData,
		MinLoanResponseRequire:   minLoanResponseRequire,
		MinCMBApprovalRequire:    minCMBApprovalRequire,
		LateWithdrawResponseFine: lateWithdrawResponseFine,
		SaleDCBTokensByUSDData:   saleDCBTokensByUSDData,
		ListLoanParams:           listLoanParams,
	}
}

func NewListSaleDataFromJson(data interface{}) []SaleData {
	listSaleDataData := common.InterfaceSlice(data)
	if listSaleDataData == nil {
		panic("list sale data")
	}
	listSaleData := make([]SaleData, 0)
	for _, i := range listSaleDataData {
		listSaleData = append(listSaleData, *NewSaleDataFromJson(i))
	}
	return listSaleData
}

func NewDCBParamsFromJson(rawData interface{}) *DCBParams {

	DCBParams := rawData.(map[string]interface{})

	listSaleData := NewListSaleDataFromJson(DCBParams["ListSaleData"])
	minLoanResponseRequire := uint8(DCBParams["MinLoanResponseRequire"].(float64))
	minCMBApprovalRequire := uint8(DCBParams["MinCMBApprovalRequire"].(float64))
	lateWithdrawResponseFine := uint64(DCBParams["LateWithdrawResponseFine"].(float64))

	saleDCBTokensByUSDData := NewSaleDCBTokensByUSDDataFromJson(DCBParams["SaleDCBTokensByUSDData"])

	listLoanParams := NewListLoanParamsFromJson(DCBParams["ListLoanParams"])
	return NewDCBParams(
		listSaleData,
		minLoanResponseRequire,
		minCMBApprovalRequire,
		lateWithdrawResponseFine,
		saleDCBTokensByUSDData,
		listLoanParams,
	)
}

type GOVParams struct {
	SalaryPerTx      uint64 // salary for each tx in block(mili constant)
	BasicSalary      uint64 // basic salary per block(mili constant)
	FeePerKbTx       uint64
	SellingBonds     *SellingBonds
	SellingGOVTokens *SellingGOVTokens
	RefundInfo       *RefundInfo
	OracleNetwork    *OracleNetwork
}

func NewGOVParams(
	salaryPerTx uint64,
	basicSalary uint64,
	feePerKbTx uint64,
	sellingBonds *SellingBonds,
	sellingGOVTokens *SellingGOVTokens,
	refundInfo *RefundInfo,
	oracleNetwork *OracleNetwork,
) *GOVParams {
	return &GOVParams{
		SalaryPerTx:      salaryPerTx,
		BasicSalary:      basicSalary,
		FeePerKbTx:       feePerKbTx,
		SellingBonds:     sellingBonds,
		SellingGOVTokens: sellingGOVTokens,
		RefundInfo:       refundInfo,
		OracleNetwork:    oracleNetwork,
	}
}

func NewGOVParamsFromJson(data interface{}) *GOVParams {
	//@todo 0xjackalope marshall
	arrayParams := data.(map[string]interface{})

	salaryPerTx := uint64(arrayParams["SalaryPerTx"].(float64))
	basicSalary := uint64(arrayParams["BasicSalary"].(float64))
	feePerKbTx := uint64(arrayParams["FeePerKbTx"].(float64))
	sellingBonds := NewSellingBondsFromJson(arrayParams["SellingBonds"])
	sellingGOVTokens := NewSellingGOVTokensFromJson(arrayParams["SellingGOVTokens"])
	refundInfo := NewRefundInfoFromJson(arrayParams["RefundInfo"])
	oracleNetwork := NewOracleNetworkFromJson(arrayParams["OracleNetwork"])

	return NewGOVParams(salaryPerTx, basicSalary, feePerKbTx, sellingBonds, sellingGOVTokens, refundInfo, oracleNetwork)
}

func (dcbParams *DCBParams) Hash() *common.Hash {
	record := ""
	for _, saleData := range dcbParams.ListSaleData {
		record = string(saleData.Hash().GetBytes())
	}
	record += string(dcbParams.SaleDCBTokensByUSDData.Hash().GetBytes())
	record += string(dcbParams.MinLoanResponseRequire)
	record += string(dcbParams.MinCMBApprovalRequire)
	record += string(dcbParams.LateWithdrawResponseFine)
	for _, i := range dcbParams.ListLoanParams {
		record += string(i.InterestRate)
		record += string(i.Maturity)
		record += string(i.LiquidationStart)
	}
	hash := common.DoubleHashH([]byte(record))
	return &hash
}

func (govParams *GOVParams) Hash() *common.Hash {
	record := string(govParams.SalaryPerTx)
	record += string(govParams.BasicSalary)
	record += string(govParams.FeePerKbTx)
	record += string(govParams.SellingBonds.Hash().GetBytes())
	record += string(govParams.SellingGOVTokens.Hash().GetBytes())
	record += string(govParams.RefundInfo.Hash().GetBytes())
	record += string(govParams.OracleNetwork.Hash().GetBytes())
	hash := common.DoubleHashH([]byte(record))
	return &hash
}

func (GOVParams GOVParams) Validate() bool {
	return true
}
func (DCBParams DCBParams) Validate() bool {
	return true
}

func (DCBParams DCBParams) ValidateSanityData() bool {
	// Todo: @0xbunyip
	return true
}

func (GOVParams GOVParams) ValidateSanityData() bool {
	// Todo: @0xankylosaurus
	return true
}
