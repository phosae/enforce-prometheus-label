package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/influxdata/promql/v2"
	"github.com/influxdata/promql/v2/pkg/labels"
	clientmodel "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
)

var (
	enforcedLabelPairs = map[string]string{
		"app":    "app",
		"region": "cn",
	}
	inputExprs = []string{
		`container_cpu_usage_seconds_total{app="",namespace="kube-system",container=~".*apiserver.*"}[5m]`,
		`container_cpu_usage_seconds_total{app="",namespace="kube-system",container=~".*apiserver.*"}`,
		`container_cpu_usage_seconds_total{namespace="kube-system",container=~".*apiserver.*"}[5m]`,
	}
	inputMetrics string = `
# HELP promhttp_metric_handler_requests_total Total number of scrapes by HTTP status code.
# TYPE promhttp_metric_handler_requests_total counter
promhttp_metric_handler_requests_total{code="200"} 0
promhttp_metric_handler_requests_total{code="500"} 0
promhttp_metric_handler_requests_total{code="503"} 0
`
)

func main() {
	// enforce label matchers for promql
	for _, rawExpr := range inputExprs {
		expr, err := promql.ParseExpr(rawExpr)
		if err != nil {
			log.Fatal(err)
		}
		injectLabelsIfNeeded(expr)
		fmt.Print(promql.Tree(expr))
	}
	fmt.Println()

	// enforce label pairs for prom metrics
	metrics, err := decodeMetric(strings.NewReader(inputMetrics), expfmt.FmtText)
	if err != nil {
		log.Fatal(err)
	}
	var pairs []*clientmodel.LabelPair
	for lk, lv := range enforcedLabelPairs {
		k, v := lk, lv
		pairs = append(pairs, &clientmodel.LabelPair{
			Name:  &k,
			Value: &v,
		})
	}
	addLabels(metrics, pairs...)
	metricEnc := expfmt.NewEncoder(os.Stdout, expfmt.FmtText)
	for i := range metrics {
		err = metricEnc.Encode(metrics[i])
		if err != nil {
			log.Fatal(err)
		}
	}
}

func injectLabelsIfNeeded(expr promql.Expr) {
	var enforce = func(inputLabels []*labels.Matcher) []*labels.Matcher {
		var provided = map[string]*labels.Matcher{}
		var ret []*labels.Matcher = inputLabels
		
		for i := range inputLabels {
			provided[inputLabels[i].Name] = inputLabels[i]
		}
		for elk, elv := range enforcedLabelPairs {
			if lm, ok := provided[elk]; ok {
				lm.Value = elv
			} else {
				ret = append(ret, &labels.Matcher{
					Type:  labels.MatchEqual,
					Name:  elk,
					Value: elv,
				})
			}
		}
		return ret
	}

	switch etyp := expr.(type) {
	case *promql.MatrixSelector:
		etyp.LabelMatchers = enforce(etyp.LabelMatchers)
	case *promql.VectorSelector:
		etyp.LabelMatchers = enforce(etyp.LabelMatchers)
	}
}

func decodeMetric(input io.Reader, format expfmt.Format) (ret []*clientmodel.MetricFamily, err error) {
	dec := expfmt.NewDecoder(input, format)

	for {
		var met clientmodel.MetricFamily
		if err = dec.Decode(&met); err == nil {
			ret = append(ret, &met)
		} else if err == io.EOF {
			return ret, nil
		} else {
			return nil, err
		}
	}
}

func addLabels(mfs []*clientmodel.MetricFamily, labels ...*clientmodel.LabelPair) {
	for _, mf := range mfs {
		for _, met := range mf.Metric {
			met.Label = append(met.Label, labels...)
		}
	}
}
