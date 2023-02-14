package controllers

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	picchu "go.medium.engineering/picchu/api/v1alpha1"
	"go.medium.engineering/picchu/controllers/garbagecollector"

	"sigs.k8s.io/controller-runtime/pkg/client"

	errorsGen "errors"

	"github.com/go-logr/logr"
)

/* Implement garbagecollectors Revision interface */

// CreatedOn returns the incarnations git timestamp
func (i *Incarnation) CreatedOn() time.Time {
	r := time.Time{}
	if i.revision != nil {
		r = i.revision.GitTimestamp()
	}
	return r
}

// State returns the incarnations current state
func (i *Incarnation) State() string {
	return string(i.status.State.Current)
}

// TTL returns the incarnations release ttl
func (i *Incarnation) TTL() time.Duration {
	r := time.Duration(0)
	if i.status != nil {
		r = time.Duration(i.status.TTL) * time.Second
	}
	return r
}

func markDeletable(ctx context.Context, log logr.Logger, cli client.Client, incarnation *Incarnation) error {
	if incarnation.revision == nil {
		return nil
	}

	key := client.ObjectKeyFromObject(incarnation.revision)
	if key == (types.NamespacedName{}) {
		return errorsGen.New("Empty NamespacedName")
	}
	// Get fresh copy of revision so there aren't any write conflicts due to concurrent revision_controller
	// TODO(bob): retry on conflict?
	revision := &picchu.Revision{}
	err := cli.Get(ctx, key, revision)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Info("Revision marked ready for deletion doesn't exist", "key", key)
			return nil
		}
		// Error reading the object - requeue the request.
		return err
	}
	label := fmt.Sprintf("%s%s", picchu.LabelTargetDeletablePrefix, incarnation.targetName())
	if revision.Labels[label] != "true" {
		log.Info("Marking revision for deletion", "revision", revision, "randomTarget", revision.Spec.Targets[0])
		revision.Labels[label] = "true"
		if err := cli.Update(ctx, revision); err != nil {
			// TODO(bob): return error when this works better
			log.Error(err, "failed to mark revision for deletion", "revision", revision)
			return nil
		}
	}
	return nil
}

func markGarbage(ctx context.Context, log logr.Logger, cli client.Client, incarnations []*Incarnation) error {
	revisions := make([]garbagecollector.Revision, len(incarnations))
	for i := range incarnations {
		revisions[i] = incarnations[i]
	}
	toDelete, err := garbagecollector.FindGarbage(log, garbagecollector.DefaultStrategy, revisions)
	if err != nil {
		return err
	}
	for i := range toDelete {
		rev := toDelete[i]
		switch inc := rev.(type) {
		case *Incarnation:
			if err = markDeletable(ctx, log, cli, inc); err != nil {
				return err
			}
		default:
			return fmt.Errorf("Unexpected revision type returned during garbage collection")
		}

	}
	return nil
}
