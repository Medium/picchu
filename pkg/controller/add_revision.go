package controller

import (
	"go.medium.engineering/picchu/pkg/controller/revision"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, revision.Add)
}
