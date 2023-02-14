package observe

import (
	"context"
	"testing"

	"github.com/go-logr/logr"

	picchu "go.medium.engineering/picchu/api/v1alpha1"
	"go.medium.engineering/picchu/mocks"
	"go.medium.engineering/picchu/test"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ReplicaSetBuilder struct {
	Tag               string
	AvailableReplicas int32
	Replicas          int32
}

func NewReplicaSetBuilder() *ReplicaSetBuilder {
	return &ReplicaSetBuilder{}
}

func (rsb *ReplicaSetBuilder) WithTag(tag string) *ReplicaSetBuilder {
	rsb.Tag = tag
	return rsb
}

func (rsb *ReplicaSetBuilder) WithAvailableReplicas(n int32) *ReplicaSetBuilder {
	rsb.AvailableReplicas = n
	return rsb
}

func (rsb *ReplicaSetBuilder) WithReplicas(n int32) *ReplicaSetBuilder {
	rsb.Replicas = n
	return rsb
}

func (rsb *ReplicaSetBuilder) Build() appsv1.ReplicaSet {
	replicas := rsb.Replicas
	return appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: rsb.Tag,
			Labels: map[string]string{
				picchu.LabelTag: rsb.Tag,
			},
		},
		Spec: appsv1.ReplicaSetSpec{
			Replicas: &replicas,
		},
		Status: appsv1.ReplicaSetStatus{
			AvailableReplicas: rsb.AvailableReplicas,
		},
	}
}

type ClientBuilder struct {
	Controller  *gomock.Controller
	ReplicaSets []appsv1.ReplicaSet
}

func NewClientBuilder() *ClientBuilder {
	return &ClientBuilder{
		ReplicaSets: []appsv1.ReplicaSet{},
	}
}

func (b *ClientBuilder) WithController(ctrl *gomock.Controller) *ClientBuilder {
	b.Controller = ctrl
	return b
}

func (b *ClientBuilder) WithReplicaSet(replicaSet appsv1.ReplicaSet) *ClientBuilder {
	b.ReplicaSets = append(b.ReplicaSets, replicaSet)
	return b
}

func (b *ClientBuilder) Build() client.Client {
	m := mocks.NewMockClient(b.Controller)
	m.
		EXPECT().
		List(gomock.Any(), mocks.InjectReplicaSets(b.ReplicaSets), gomock.Any()).
		Return(nil).
		AnyTimes()
	return m
}

type ClusterObserverBuilder struct {
	Name          string
	Live          bool
	ClientBuilder ClientBuilder
	Log           logr.Logger
}

func NewClusterObserverBuilder() *ClusterObserverBuilder {
	return &ClusterObserverBuilder{}
}

func (c *ClusterObserverBuilder) WithName(name string) *ClusterObserverBuilder {
	c.Name = name
	return c
}

func (c *ClusterObserverBuilder) WithLiveness(liveness bool) *ClusterObserverBuilder {
	c.Live = liveness
	return c
}

func (c *ClusterObserverBuilder) WithTest(t *testing.T) *ClusterObserverBuilder {
	c.ClientBuilder.WithController(gomock.NewController(t))
	return c
}

func (c *ClusterObserverBuilder) WithReplicaSet(tag string, availableReplicas int32, replicas int32) *ClusterObserverBuilder {
	c.ClientBuilder.WithReplicaSet(NewReplicaSetBuilder().
		WithTag(tag).
		WithAvailableReplicas(availableReplicas).
		WithReplicas(replicas).
		Build(),
	)
	return c
}

func (c *ClusterObserverBuilder) WithLog(log logr.Logger) *ClusterObserverBuilder {
	c.Log = log
	return c
}

func (c *ClusterObserverBuilder) Build() Observer {
	return NewClusterObserver(Cluster{Name: c.Name, Live: c.Live}, c.ClientBuilder.Build(), c.Log)
}

func (c *ClusterObserverBuilder) Finish() {
	if c.ClientBuilder.Controller != nil {
		c.ClientBuilder.Controller.Finish()
	}
}

func TestObservation(t *testing.T) {
	ctx := context.TODO()
	log := test.MustNewLogger()
	observers := []Observer{}

	o := NewClusterObserverBuilder().
		WithTest(t).
		WithLiveness(true).
		WithLog(log).
		WithName("cluster0").
		WithReplicaSet("rs0", 1, 1).
		WithReplicaSet("rs1", 0, 2)
	defer o.Finish()
	observers = append(observers, o.Build())

	o = NewClusterObserverBuilder().
		WithTest(t).
		WithLiveness(true).
		WithLog(log).
		WithName("cluster1").
		WithReplicaSet("rs0", 0, 1).
		WithReplicaSet("rs1", 2, 2)
	defer o.Finish()
	observers = append(observers, o.Build())

	o = NewClusterObserverBuilder().
		WithTest(t).
		WithLiveness(true).
		WithLog(log).
		WithName("cluster2").
		WithReplicaSet("rs0", 0, 1).
		WithReplicaSet("rs1", 1, 2)
	defer o.Finish()
	observers = append(observers, o.Build())

	o = NewClusterObserverBuilder().
		WithTest(t).
		WithLiveness(true).
		WithLog(log).
		WithName("cluster3").
		WithReplicaSet("rs0", 1, 1).
		WithReplicaSet("rs1", 2, 2)
	defer o.Finish()
	observers = append(observers, o.Build())

	o = NewClusterObserverBuilder().
		WithTest(t).
		WithLiveness(false).
		WithLog(log).
		WithName("standby0").
		WithReplicaSet("rs0", 1, 1).
		WithReplicaSet("rs1", 2, 2)
	defer o.Finish()
	observers = append(observers, o.Build())

	observer := NewConcurrentObserver(observers, log)
	observation, _ := observer.Observe(ctx, "test-namespace")

	info := observation.InfoForTag("rs0")
	assert.Equal(t, 4, int(info.Live.Current.Count))
	assert.Equal(t, 2, int(info.Live.Current.Sum))
	assert.Equal(t, 0, int(info.Live.Current.Min))
	assert.Equal(t, 1, int(info.Live.Current.Max))
	assert.Equal(t, 4, int(info.Live.Desired.Count))
	assert.Equal(t, 4, int(info.Live.Desired.Sum))
	assert.Equal(t, 1, int(info.Live.Desired.Min))
	assert.Equal(t, 1, int(info.Live.Desired.Max))
	assert.Equal(t, 1, int(info.Standby.Current.Count))
	assert.Equal(t, 1, int(info.Standby.Current.Sum))
	assert.Equal(t, 1, int(info.Standby.Current.Min))
	assert.Equal(t, 1, int(info.Standby.Current.Max))
	assert.Equal(t, 1, int(info.Standby.Desired.Count))
	assert.Equal(t, 1, int(info.Standby.Desired.Sum))
	assert.Equal(t, 1, int(info.Standby.Desired.Min))
	assert.Equal(t, 1, int(info.Standby.Desired.Max))

	info = observation.InfoForTag("rs1")
	assert.Equal(t, 4, int(info.Live.Current.Count))
	assert.Equal(t, 5, int(info.Live.Current.Sum))
	assert.Equal(t, 0, int(info.Live.Current.Min))
	assert.Equal(t, 2, int(info.Live.Current.Max))
	assert.Equal(t, 4, int(info.Live.Desired.Count))
	assert.Equal(t, 8, int(info.Live.Desired.Sum))
	assert.Equal(t, 2, int(info.Live.Desired.Min))
	assert.Equal(t, 2, int(info.Live.Desired.Max))
	assert.Equal(t, 1, int(info.Standby.Current.Count))
	assert.Equal(t, 2, int(info.Standby.Current.Sum))
	assert.Equal(t, 2, int(info.Standby.Current.Min))
	assert.Equal(t, 2, int(info.Standby.Current.Max))
	assert.Equal(t, 1, int(info.Standby.Desired.Count))
	assert.Equal(t, 2, int(info.Standby.Desired.Sum))
	assert.Equal(t, 2, int(info.Standby.Desired.Min))
	assert.Equal(t, 2, int(info.Standby.Desired.Max))
}
