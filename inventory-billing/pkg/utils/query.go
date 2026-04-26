package utils

import (
	"fmt"
	"strings"
)

// OrderClause builds a safe "column DIR" ORDER BY string from user-supplied
// sort parameters. Only columns present in allowed are accepted; everything
// else falls back to defaultCol. Direction is normalised to ASC or DESC.
//
// Usage (in any repository):
//
//	order := utils.OrderClause(filter.SortBy, filter.SortDir,
//	    []string{"name", "price", "stock", "created_at"}, "created_at")
func OrderClause(sortBy, sortDir string, allowed []string, defaultCol string) string {
	col := defaultCol
	for _, a := range allowed {
		if a == sortBy {
			col = sortBy
			break
		}
	}

	dir := "DESC"
	if strings.EqualFold(sortDir, "asc") {
		dir = "ASC"
	}
	return fmt.Sprintf("%s %s", col, dir)
}
