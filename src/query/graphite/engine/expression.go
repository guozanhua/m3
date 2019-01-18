package engine

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/m3db/m3/src/query/graphite/querycontext"
	"github.com/m3db/m3/src/query/graphite/ts"
)

var (
	errTopLevelFunctionMustReturnTimeSeries = errors.New("top-level functions must return timeseries data")
)

// An Expression is a metric query expression
type Expression interface {
	CallASTNode
	// Executes the expression against the given context, and returns the resulting time series data
	Execute(ctx *querycontext.Context) (ts.SeriesList, error)
}

// CallASTNode is an interface to help with printing the AST.
type CallASTNode interface {
	// Name returns the name of the call.
	Name() string
	// Arguments describe each argument that the call has, some
	// arguments can be casted to an Call themselves.
	Arguments() []ArgumentASTNode
}

// ArgumentASTNode is an interface to help with printing the AST.
type ArgumentASTNode interface {
	String() string
}

// A fetchExpression is an expression that fetches a bunch of data from storage based on a path expression
type fetchExpression struct {
	// The path expression to fetch
	pathArg fetchExpressionPathArg
}

type fetchExpressionPathArg struct {
	path string
}

func (a fetchExpressionPathArg) String() string {
	return a.path
}

// newFetchExpression creates a new fetch expression for a single path
func newFetchExpression(path string) *fetchExpression {
	return &fetchExpression{pathArg: fetchExpressionPathArg{path: path}}
}

func (f *fetchExpression) Name() string {
	return "fetch"
}

func (f *fetchExpression) Arguments() []ArgumentASTNode {
	return []ArgumentASTNode{f.pathArg}
}

// Execute fetches results from storage
func (f *fetchExpression) Execute(ctx *querycontext.Context) (ts.SeriesList, error) {
	result, err := ctx.Engine.FetchByQuery(ctx, f.pathArg.path, ctx.StartTime, ctx.EndTime,
		ctx.LocalOnly, ctx.UseCache, ctx.UseM3DB, ctx.Timeout)
	if err != nil {
		return ts.SeriesList{}, err
	}

	for _, r := range result.SeriesList {
		r.Specification = f.pathArg.path
	}
	return ts.SeriesList{Values: result.SeriesList}, nil
}

// Evaluate evaluates the fetch and returns its results as a reflection value, allowing it to be used
// as an input argument to a function that takes a time series
func (f *fetchExpression) Evaluate(ctx *querycontext.Context) (reflect.Value, error) {
	timeseries, err := f.Execute(ctx)
	if err != nil {
		return reflect.Value{}, err
	}

	return reflect.ValueOf(timeseries), nil
}

// CompatibleWith returns true if the reflected type is a time series or a generic interface.
func (f *fetchExpression) CompatibleWith(reflectType reflect.Type) bool {
	return reflectType == singlePathSpecType || reflectType == multiplePathSpecsType || reflectType == interfaceType
}

func (f *fetchExpression) String() string {
	return fmt.Sprintf("fetch(%s)", f.pathArg.path)
}

// A funcExpression is an expression that evaluates a function returning a timeseries
type funcExpression struct {
	call *functionCall
}

// newFuncExpression creates a new expressioon based on the given function call
func newFuncExpression(call *functionCall) (Expression, error) {
	if !(call.f.out == seriesListType || call.f.out == unaryContextShifterPtrType || call.f.out == binaryContextShifterPtrType) {
		return nil, errTopLevelFunctionMustReturnTimeSeries
	}

	return &funcExpression{call: call}, nil
}

func (f *funcExpression) Name() string {
	return f.call.Name()
}

func (f *funcExpression) Arguments() []ArgumentASTNode {
	return f.call.Arguments()
}

// Execute evaluates the function and returns the result as a timeseries
func (f *funcExpression) Execute(ctx *querycontext.Context) (ts.SeriesList, error) {
	out, err := f.call.Evaluate(ctx)
	if err != nil {
		return ts.SeriesList{}, err
	}

	return out.Interface().(ts.SeriesList), nil
}

func (f *funcExpression) String() string { return f.call.String() }

// A noopExpression is an empty expression that returns nothing
type noopExpression struct{}

// Execute returns nothing
func (noop noopExpression) Execute(ctx *querycontext.Context) (ts.SeriesList, error) {
	return ts.SeriesList{}, nil
}

func (noop noopExpression) Name() string {
	return "noop"
}

func (noop noopExpression) Arguments() []ArgumentASTNode {
	return nil
}

func (noop noopExpression) String() string {
	return noop.Name()
}
