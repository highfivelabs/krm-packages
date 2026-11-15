// Copyright 2026 High Five Labs
// SPDX-License-Identifier: Apache-2.0

package convert

import (
	"fmt"

	"github.com/highfivelabs/catalog/tools/upstream"
)

// Refusals applies a chart's refusal of a variable to a value, as the
// converter does, for callers outside it (the factory generator, package
// factory). v is plain data (map[string]any, []any, scalars). It returns
// what cannot be carried, one line each ("where: refused by the chart
// ..."), and whether the value can be carried at all.
func Refusals(r upstream.Refused, v any, where string, allowAlpha bool) (gaps []string, carried bool) {
	carried = checkRefused(r, v, where, allowAlpha, func(level, w, format string, a ...any) {
		if level == Gap {
			gaps = append(gaps, w+": "+fmt.Sprintf(format, a...))
		}
	})
	return gaps, carried
}
