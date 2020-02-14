package prometheus

import (
	"errors"

	"github.com/prometheus/prometheus/promql"
)

// MetricNames returns a map (set of keys) of unique metric names included in PromQL query string
func MetricNames(query string) (map[string]bool, error) {
	expr, err := promql.ParseExpr(query)
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

func metricNames(m map[string]bool, node promql.Node) error {
	switch n := node.(type) {
	case *promql.EvalStmt:
		metricNames(m, n.Expr)

	case promql.Expressions:
		for _, e := range n {
			metricNames(m, e)
		}
	case *promql.AggregateExpr:
		metricNames(m, n.Expr)

	case *promql.BinaryExpr:
		metricNames(m, n.LHS)
		metricNames(m, n.RHS)

	case *promql.Call:
		metricNames(m, n.Args)

	case *promql.ParenExpr:
		metricNames(m, n.Expr)

	case *promql.UnaryExpr:
		metricNames(m, n.Expr)

	case *promql.SubqueryExpr:
		metricNames(m, n.Expr)

	case *promql.VectorSelector:
		m[n.Name] = true

	case *promql.MatrixSelector:
		m[n.Name] = true

	case *promql.NumberLiteral, *promql.StringLiteral:
	// nothing to do

	default:
		return errors.New("promql.getMetrics: not all node types covered")
	}

	return nil
}
