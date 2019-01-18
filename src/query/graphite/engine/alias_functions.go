package engine

import (
	"github.com/m3db/m3/src/query/graphite/querycontext"
	"github.com/m3db/m3/src/query/graphite/ts"
)

// alias takes one metric or a wildcard seriesList and a string in quotes.
// Prints the string instead of the metric name in the legend.
func alias(ctx *querycontext.Context, series singlePathSpec, a string) (ts.SeriesList, error) {
	return querycontext.Alias(ctx, ts.SeriesList(series), a)
}

// aliasByMetric takes a seriesList and applies an alias derived from the base
// metric name.
func aliasByMetric(ctx *querycontext.Context, series singlePathSpec) (ts.SeriesList, error) {
	return querycontext.AliasByMetric(ctx, ts.SeriesList(series))
}

// aliasByNode renames a time series result according to a subset of the nodes
// in its hierarchy.
func aliasByNode(ctx *querycontext.Context, seriesList singlePathSpec, nodes ...int) (ts.SeriesList, error) {
	return querycontext.AliasByNode(ctx, ts.SeriesList(seriesList), nodes...)
}

// aliasSub runs series names through a regex search/replace.
func aliasSub(ctx *querycontext.Context, input singlePathSpec, search, replace string) (ts.SeriesList, error) {
	return querycontext.AliasSub(ctx, ts.SeriesList(input), search, replace)
}
