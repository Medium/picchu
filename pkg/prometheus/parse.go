package prometheus

import (
	"errors"
	//promql "github.com/prometheus/prometheus/promql"
	parser "github.com/prometheus/prometheus/promql/parser"
)

// MetricNames returns a map (set of keys) of unique metric names included in PromQL query string
func MetricNames(query string) (map[string]bool, error) {

	expr, err := parser.ParseExpr(query)
	if err != nil {
		return nil, err
	}

	m := make(map[string]bool)
	err = metricNames(m, expr)
	if err != nil {
		return nil, err
	}

	return m, nil
}

func metricNames(m map[string]bool, node parser.Node) error {
	switch n := node.(type) {
	case *parser.EvalStmt:
		metricNames(m, n.Expr)

	case parser.Expressions:
		for _, e := range n {
			metricNames(m, e)
		}
	case *parser.AggregateExpr:
		metricNames(m, n.Expr)

	case *parser.SubqueryExpr:
		metricNames(m, n.Expr)

	case *parser.BinaryExpr:
		metricNames(m, n.LHS)
		metricNames(m, n.RHS)

	case *parser.Call:
		metricNames(m, n.Args)

	case *parser.ParenExpr:
		metricNames(m, n.Expr)

	case *parser.UnaryExpr:
		metricNames(m, n.Expr)

	case *parser.VectorSelector:
		m[n.Name] = true

	case *parser.MatrixSelector:
		metricNames(m, n.VectorSelector)
		//m[n.Name] = true

	case *parser.NumberLiteral, *parser.StringLiteral:
	// nothing to do

	default:
		return errors.New("promql.getMetrics: not all node types covered")
	}

	return nil
}
