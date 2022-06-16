package mocks

import (
	"bytes"
	"fmt"

	"github.com/golang/mock/gomock"
	"k8s.io/apimachinery/pkg/runtime"
	types "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type callback struct {
	fn   func(x interface{}) bool
	desc string
}

func (m *callback) Matches(x interface{}) bool {
	return m.fn(x)
}

func (m *callback) String() string {
	return m.desc
}

// Callback saves you the toil of creating a struct for a matcher
func Callback(fn func(x interface{}) bool, desc string) gomock.Matcher {
	return &callback{fn, desc}
}

// ListOptions matches client.ListOptions
func ListOptions(opts *client.ListOptions) gomock.Matcher {
	fn := func(x interface{}) bool {
		switch o := x.(type) {
		case *client.ListOptions:
			if opts.LabelSelector != nil && o.LabelSelector != nil {
				if opts.LabelSelector.String() != o.LabelSelector.String() {
					return false
				}
			} else {
				if opts.LabelSelector != o.LabelSelector {
					return false
				}
			}
			if opts.FieldSelector != nil && o.FieldSelector != nil {
				if opts.FieldSelector.String() != o.FieldSelector.String() {
					return false
				}
			} else {
				if opts.FieldSelector != o.FieldSelector {
					return false
				}
			}
			return opts.Namespace == o.Namespace
		default:
			return false
		}
	}
	return Callback(fn, fmt.Sprintf("matches %#v", *opts))
}

// Kind matches k8s kinds
func Kind(k string) gomock.Matcher {
	fn := func(x interface{}) bool {
		switch o := x.(type) {
		case runtime.Object:
			kinds, _, _ := scheme.Scheme.ObjectKinds(o)
			for _, kind := range kinds {
				if kind.Kind == k {
					return true
				}
			}
			return false
		default:
			return false
		}
	}
	return Callback(fn, fmt.Sprintf("matches kind %s", k))
}

// NamespacedName matches a k8s objects namespace and name
func NamespacedName(namespace, name string) gomock.Matcher {
	fn := func(x interface{}) bool {
		switch o := x.(type) {
		case client.Object:
			key := client.ObjectKeyFromObject(o)
			if key == (types.NamespacedName{}) {
				return false
			}
			return key.Namespace == namespace &&
				key.Name == name
		default:
			return false
		}
	}
	return Callback(fn, fmt.Sprintf("matches %s/%s", namespace, name))
}

// And combines matchers
func And(matchers ...gomock.Matcher) gomock.Matcher {
	fn := func(x interface{}) bool {
		for _, m := range matchers {
			if !m.Matches(x) {
				return false
			}
		}
		return true
	}
	s := bytes.NewBufferString("and(")
	first := true
	for _, m := range matchers {
		if !first {
			s.WriteString(", ")
		}
		first = false
		s.WriteString(m.String())
	}
	s.WriteString(")")
	return Callback(fn, s.String())
}

// ObjectKey matches an argument to an ObjectKey
func ObjectKey(ok client.ObjectKey) gomock.Matcher {
	fn := func(x interface{}) bool {
		switch o := x.(type) {
		case client.ObjectKey:
			return o.Name == ok.Name &&
				o.Namespace == ok.Namespace
		default:
			return false
		}
	}
	return Callback(fn, fmt.Sprintf("matches objectkey %s", ok))
}
