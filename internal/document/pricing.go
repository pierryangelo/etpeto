package document

import (
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"
)

// Price research is the first structured element of the ETP. It is not rich
// text with a table drawn inside it: it is data the document owns, validated
// here and rendered by the template.
const ElementPriceResearch = "priceResearch"

// Estimation methods. Which one was used is stored with the element, because a
// price study has to state how it reached its number.
const (
	MethodAverage = "average"
	MethodMedian  = "median"
	MethodLowest  = "lowest"
)

const (
	maxPriceItems       = 200
	maxQuotesPerItem    = 50
	maxDescriptionRunes = 300
	maxUnitRunes        = 40
	maxSourceRunes      = 200
	maxReferenceRunes   = 500
	// A quantity is a count of goods or services, not a measurement, so it is
	// kept whole. Fractional needs (hours, litres) belong to the unit.
	maxQuantity = 1_000_000
	// One billion in cents: far above any plausible unit price, and low enough
	// that totals cannot overflow int64.
	maxUnitPriceCents = 100_000_000_000
)

var validMethods = map[string]bool{
	MethodAverage: true,
	MethodMedian:  true,
	MethodLowest:  true,
}

// PriceQuote is one observation of a price, with where it came from and when.
// A study without sources and dates is not verifiable.
type PriceQuote struct {
	ID             string `json:"id"`
	Source         string `json:"source"`
	Reference      string `json:"reference,omitempty"`
	ObservedOn     string `json:"observedOn"`
	UnitPriceCents int64  `json:"unitPriceCents"`
}

type PriceItem struct {
	ID          string       `json:"id"`
	Description string       `json:"description"`
	Unit        string       `json:"unit"`
	Quantity    int64        `json:"quantity"`
	Quotes      []PriceQuote `json:"quotes"`
}

type PriceResearch struct {
	Items         []PriceItem `json:"items"`
	Method        string      `json:"method"`
	Justification string      `json:"justification,omitempty"`
}

// PriceEstimate is derived, never stored: recomputing it from the quotes is
// what keeps the number honest when a quote changes.
type PriceEstimate struct {
	UnitPriceCents  int64
	TotalCents      int64
	ConsideredCount int
}

func (r PriceResearch) Validate() error {
	if !validMethods[r.Method] {
		return fmt.Errorf("%w: unsupported estimation method", ErrInvalidDocument)
	}
	if len(r.Items) > maxPriceItems {
		return fmt.Errorf("%w: price item count", ErrInvalidDocument)
	}
	seen := make(map[string]bool, len(r.Items))
	for index, item := range r.Items {
		if err := item.validate(seen); err != nil {
			return fmt.Errorf("%w in item %d", err, index)
		}
	}
	return nil
}

func (i PriceItem) validate(seen map[string]bool) error {
	if i.ID == "" || seen[i.ID] {
		return fmt.Errorf("%w: item identifier", ErrInvalidDocument)
	}
	seen[i.ID] = true
	if err := boundedText("item description", i.Description, maxDescriptionRunes); err != nil {
		return err
	}
	if utf8.RuneCountInString(i.Unit) > maxUnitRunes {
		return fmt.Errorf("%w: item unit", ErrInvalidDocument)
	}
	if i.Quantity < 0 || i.Quantity > maxQuantity {
		return fmt.Errorf("%w: item quantity", ErrInvalidDocument)
	}
	if len(i.Quotes) > maxQuotesPerItem {
		return fmt.Errorf("%w: quote count", ErrInvalidDocument)
	}
	for _, quote := range i.Quotes {
		if err := quote.validate(seen); err != nil {
			return err
		}
	}
	return nil
}

func (q PriceQuote) validate(seen map[string]bool) error {
	if q.ID == "" || seen[q.ID] {
		return fmt.Errorf("%w: quote identifier", ErrInvalidDocument)
	}
	seen[q.ID] = true
	if err := boundedText("quote source", q.Source, maxSourceRunes); err != nil {
		return err
	}
	if utf8.RuneCountInString(q.Reference) > maxReferenceRunes {
		return fmt.Errorf("%w: quote reference", ErrInvalidDocument)
	}
	if q.UnitPriceCents < 0 || q.UnitPriceCents > maxUnitPriceCents {
		return fmt.Errorf("%w: quote unit price", ErrInvalidDocument)
	}
	if q.ObservedOn != "" && !isISODate(q.ObservedOn) {
		return fmt.Errorf("%w: quote observation date", ErrInvalidDocument)
	}
	return nil
}

// Estimate applies the recorded method to an item's quotes. It returns the
// unit price and the total, both in cents, along with how many quotes were
// considered — a number the document has to be able to state.
func (r PriceResearch) Estimate(item PriceItem) PriceEstimate {
	prices := make([]int64, 0, len(item.Quotes))
	for _, quote := range item.Quotes {
		prices = append(prices, quote.UnitPriceCents)
	}
	if len(prices) == 0 {
		return PriceEstimate{}
	}
	sort.Slice(prices, func(a, b int) bool { return prices[a] < prices[b] })

	var unit int64
	switch r.Method {
	case MethodLowest:
		unit = prices[0]
	case MethodMedian:
		middle := len(prices) / 2
		if len(prices)%2 == 1 {
			unit = prices[middle]
		} else {
			unit = roundedMean(prices[middle-1] + prices[middle])
		}
	default:
		var sum int64
		for _, price := range prices {
			sum += price
		}
		unit = divideRounded(sum, int64(len(prices)))
	}
	return PriceEstimate{
		UnitPriceCents:  unit,
		TotalCents:      unit * item.Quantity,
		ConsideredCount: len(prices),
	}
}

// Total sums every item's estimate. Summing the rounded item totals — rather
// than rounding once at the end — is what makes the printed table add up to the
// printed total.
func (r PriceResearch) Total() int64 {
	var total int64
	for _, item := range r.Items {
		total += r.Estimate(item).TotalCents
	}
	return total
}

func roundedMean(sum int64) int64 {
	return divideRounded(sum, 2)
}

// divideRounded rounds half away from zero, the rule used for currency in
// Brazilian public price studies, so the result never depends on the language's
// default rounding mode.
func divideRounded(sum int64, count int64) int64 {
	if count == 0 {
		return 0
	}
	if sum >= 0 {
		return (sum + count/2) / count
	}
	return -((-sum + count/2) / count)
}

func boundedText(field string, value string, limit int) error {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" || utf8.RuneCountInString(trimmed) > limit {
		return fmt.Errorf("%w: %s", ErrInvalidDocument, field)
	}
	return nil
}

func isISODate(value string) bool {
	if len(value) != 10 || value[4] != '-' || value[7] != '-' {
		return false
	}
	for index, character := range value {
		if index == 4 || index == 7 {
			continue
		}
		if character < '0' || character > '9' {
			return false
		}
	}
	return true
}
