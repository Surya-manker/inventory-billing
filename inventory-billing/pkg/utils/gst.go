package utils

import "strings"

// GSTBreakdown holds the computed tax split for an invoice.
// Either (CGSTAmount + SGSTAmount) or IGSTAmount is non-zero — never both.
type GSTBreakdown struct {
	IntraState  bool
	CGSTRate    float64
	SGSTRate    float64
	IGSTRate    float64
	CGSTAmount  float64
	SGSTAmount  float64
	IGSTAmount  float64
	TotalAmount float64
}

// IsIntraState returns true when the seller and buyer share the same GST state code.
// Defaults to intra-state when either code is absent (covers B2C and unregistered buyers).
func IsIntraState(sellerStateCode, customerGSTIN string) bool {
	if sellerStateCode == "" || len(customerGSTIN) < 2 {
		return true
	}
	return strings.EqualFold(customerGSTIN[:2], sellerStateCode)
}

// SplitGST produces the CGST/SGST or IGST breakdown from a pre-computed total tax amount.
// The split is always 50/50 for CGST+SGST; IGST is the full amount.
func SplitGST(totalTax, taxRate float64, intraState bool) GSTBreakdown {
	b := GSTBreakdown{
		IntraState:  intraState,
		TotalAmount: totalTax,
	}
	if intraState {
		half := Round2(totalTax / 2)
		b.CGSTRate = Round2(taxRate / 2)
		b.SGSTRate = Round2(taxRate / 2)
		b.CGSTAmount = half
		b.SGSTAmount = Round2(totalTax - half) // absorbs the odd-penny remainder
	} else {
		b.IGSTRate = taxRate
		b.IGSTAmount = totalTax
	}
	return b
}
