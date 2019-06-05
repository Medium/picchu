package plan

import (
	"fmt"
	"reflect"

	"go.medium.engineering/picchu/pkg/controller/releasemanager/mocks"

	"github.com/golang/mock/gomock"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

var (
	notFoundError = &errors.StatusError{metav1.Status{
		Reason: metav1.StatusReasonNotFound,
	}}
)

type replicaSetMatcher func(*appsv1.ReplicaSet) bool

func replicaSetCallback(fn replicaSetMatcher) gomock.Matcher {
	return mocks.Callback(func(x interface{}) bool {
		switch o := x.(type) {
		case *appsv1.ReplicaSet:
			return fn(o)
		default:
			return false
		}
	}, "replicaSet callback")
}

func mustParseQuantity(val string) resource.Quantity {
	r, err := resource.ParseQuantity(val)
	if err != nil {
		panic("Failed to parse Quantity")
	}
	return r
}

func k8sEqual(expected runtime.Object) gomock.Matcher {
	return mocks.Callback(func(x interface{}) bool {
		switch o := x.(type) {
		case runtime.Object:
			return reflect.DeepEqual(expected, o)
		default:
			return false
		}
	}, fmt.Sprintf("matches %v", expected))
}
