package plan

import (
	"context"
	_ "runtime"
	"testing"

	testify "github.com/stretchr/testify/assert"
	ktest "go.medium.engineering/kubernetes/pkg/test"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV1"
	ddog "github.com/DataDog/datadog-operator/api/datadoghq/v1alpha1"
	picchu "go.medium.engineering/picchu/api/v1alpha1"
	picchuv1alpha1 "go.medium.engineering/picchu/api/v1alpha1"
	"go.medium.engineering/picchu/test"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestDeleteSpecificDatadogMonitors(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	assert := testify.New(t)
	log := test.MustNewLogger()
	toRemove := []datadogV1.SearchServiceLevelObjective{}
	sslo := datadogV1.SearchServiceLevelObjective{}
	attr := datadogV1.SearchServiceLevelObjectiveAttributes{}
	name := "test"
	attr.Name = &name
	data := datadogV1.SearchServiceLevelObjectiveData{
		Attributes: &attr,
	}
	sslo.Data = &data
	toRemove = append(toRemove, sslo)
	deleteSpecificDatadogMonitors := &DeleteSpecificDatadogMonitors{
		App:       "testapp",
		Namespace: "testnamespace",
		ToRemove:  toRemove,
	}
	pr := &ddog.DatadogMonitor{
		ObjectMeta: meta.ObjectMeta{
			Name:      "test",
			Namespace: "testnamespace",
			Labels: map[string]string{
				picchu.LabelApp:                 deleteSpecificDatadogMonitors.App,
				picchuv1alpha1.LabelMonitorType: MonitorTypeSLO,
				picchuv1alpha1.LabelSLOName:     *toRemove[0].Data.Attributes.Name,
			},
		},
	}
	cli := fakeClient(pr)

	assert.NoError(deleteSpecificDatadogMonitors.Apply(ctx, cli, cluster, log), "Shouldn't return error.")
	ktest.AssertNotFound(ctx, t, cli, pr)
}
