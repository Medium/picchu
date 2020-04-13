package plan

import (
	"bytes"
	"fmt"
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"

	"go.medium.engineering/picchu/pkg/controller/utils"
	"go.medium.engineering/picchu/pkg/mocks"

	"github.com/andreyvit/diff"
	"github.com/golang/mock/gomock"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
)

var (
	NotFoundError = &errors.StatusError{metav1.Status{
		Reason: metav1.StatusReasonNotFound,
	}}
	yamlSerializer runtime.Serializer
)

func init() {
	yamlSerializer = json.NewYAMLSerializer(json.DefaultMetaFactory, nil, nil)
}

type ReplicaSetMatcher func(*appsv1.ReplicaSet) bool

func ReplicaSetCallback(fn ReplicaSetMatcher) gomock.Matcher {
	return mocks.Callback(func(x interface{}) bool {
		switch o := x.(type) {
		case *appsv1.ReplicaSet:
			return fn(o)
		default:
			return false
		}
	}, "replicaSet callback")
}

func encodeResource(o runtime.Object) (string, error) {
	buf := &bytes.Buffer{}
	if err := yamlSerializer.Encode(o, buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func ResourcesEqual(t *testing.T, expected, actual runtime.Object, msgAndArgs ...interface{}) {
	eYaml, err := encodeResource(expected)
	assert.NoError(t, err)
	aYaml, err := encodeResource(actual)
	assert.NoError(t, err)
	assert.Equal(t, eYaml, aYaml, msgAndArgs...)
}

func resourceDiff(expected, actual runtime.Object) (string, error) {
	eYaml, err := encodeResource(expected)
	if err != nil {
		return "", err
	}
	aYaml, err := encodeResource(actual)
	if err != nil {
		return "", err
	}
	return diff.LineDiff(eYaml, aYaml), nil
}

func K8sEqual(expected runtime.Object) gomock.Matcher {
	description := "nil"
	if expected != nil {
		yaml, err := encodeResource(expected)
		if err == nil {
			description = fmt.Sprintf("matches %v", yaml)
		} else {
			description = "error printing object"
		}
	}
	return mocks.Callback(func(x interface{}) bool {
		switch o := x.(type) {
		case runtime.Object:
			r := reflect.DeepEqual(expected, o)
			if !r && utils.MustGetKind(expected) == utils.MustGetKind(o) {
				diff, err := resourceDiff(expected, o)
				if err != nil {
					panic(err)
				}
				fmt.Println("k8sEqual comparison failed with:")
				fmt.Println(diff)
			}
			return r
		default:
			return false
		}
	}, description)
}
