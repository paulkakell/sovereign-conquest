package game

import "math"

func PricePerUnit(basePrice, baseQty, qty int) int {
	if basePrice < 1 {
		basePrice = 1
	}
	if baseQty <= 0 {
		return basePrice
	}
	if qty < 0 {
		qty = 0
	}
	if qty > baseQty {
		qty = baseQty
	}

	scarcity := float64(baseQty-qty) / float64(baseQty) // 0..1
	mult := 1.0 + scarcity                              // 1.0..2.0
	price := int(math.Round(float64(basePrice) * mult))
	if price < 1 {
		price = 1
	}
	return price
}

// PricePerUnitWithPercent applies a simple price multiplier (percentage) to basePrice before
// calculating scarcity-based pricing.
//
// percent is clamped to a reasonable range to prevent extreme or negative prices.
func PricePerUnitWithPercent(basePrice, baseQty, qty int, percent int) int {
	if percent < 10 {
		percent = 10
	}
	if percent > 300 {
		percent = 300
	}
	if percent != 100 {
		basePrice = int(math.Round(float64(basePrice) * float64(percent) / 100.0))
	}
	return PricePerUnit(basePrice, baseQty, qty)
}
