package controller

import (
	"go.medium.engineering/picchu/pkg/controller/utils"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// AddToManagerFuncs is a list of functions to add all Controllers to the Manager
var AddToManagerFuncs []func(manager.Manager, utils.Config) error

// AddToManager adds all Controllers to the Manager
func AddToManager(m manager.Manager, c utils.Config) error {
	for _, f := range AddToManagerFuncs {
		if err := f(m, c); err != nil {
			return err
		}
	}
	return nil
}
