package prometheus

import (
	"reflect"
	"testing"
)

func TestMetricNames(t *testing.T) {
	queries := []struct {
		query    string
		expected map[string]bool
	}{
		{"1", map[string]bool{}},
		{"sum(metrics_test)by(le)", map[string]bool{"metrics_test": true}},
		{"max_over_time( deriv( rate(metrics_test[1m])[5m:1m] )[10m:] )", map[string]bool{"metrics_test": true}},
		{"sum(metrics_test{service_name!~\"\"}) by (label1, test)", map[string]bool{"metrics_test": true}},
		{"metrics_test", map[string]bool{"metrics_test": true}},
		{"sum(rate(metrics_test{service_name!~\"\"}[5m])) by (label1, test)", map[string]bool{"metrics_test": true}},
		{"metrics_test{methodName=~\"label1|label2\"}", map[string]bool{"metrics_test": true}},
		{"sum(rate(metrics_test{label1=~\".*Test\",label2=~\"test\"}[5m])) by (label1)", map[string]bool{"metrics_test": true}},
		{"sum(count_over_time(metrics_test[5m])) by (label1)", map[string]bool{"metrics_test": true}},
		{"(avg(metrics_test) by (label1))", map[string]bool{"metrics_test": true}},
		{"topk(20, metrics_test{label1=~\"test1\", label2=~\"test2\"})", map[string]bool{"metrics_test": true}},
		{"sum(rate(metrics_test{label1=~\".*Test\",label2=~\"test\"}[5m])) by (label1) / " +
			"sum(rate(metrics_test2{label1=~\".*Test\",label2=~\"test\"}[5m])) by (label1)", map[string]bool{"metrics_test": true, "metrics_test2": true},
		},
		{"sum(rate(metrics_test{label1=~\".*Test\",label2=~\"test\"}[5m])) by (label1) / " +
			"sum(rate(metrics_test{label1=~\".*Test\",label2=~\"test\"}[5m])) by (label1)", map[string]bool{"metrics_test": true},
		},
		{"sum(rate(metrics_test{label1=~\".*Test\",label2=~\"test\"}[5m])) by (label1) * " +
			"sum(rate(metrics_test2{label1=~\".*Test\",label2=~\"test\"}[5m])) by (label1) + " +
			"sum(rate(metrics_test3{label1=~\".*Test\",label2=~\"test\"}[5m])) by (label1) / " +
			"sum(rate(metrics_test4{label1=~\".*Test\",label2=~\"test\"}[5m])) by (label1) - " +
			"sum(rate(metrics_test5{label1=~\".*Test\",label2=~\"test\"}[5m])) by (label1)",
			map[string]bool{"metrics_test": true, "metrics_test2": true, "metrics_test3": true, "metrics_test4": true, "metrics_test5": true},
		},
	}

	for _, query := range queries {
		actual, err := MetricNames(query.query)
		if err != nil {
			t.Error(err)
			t.Fail()
		} else {
			if !reflect.DeepEqual(query.expected, actual) {
				t.Errorf("Expected did not match actual. Expected: %v. Actual: %v", query.expected, actual)
				t.Fail()
			}
		}
	}
}
