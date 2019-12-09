package releasemanager

import (
	"context"
	"fmt"
	"time"

	picchu "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"
	"go.medium.engineering/picchu/pkg/controller/releasemanager/garbagecollector"

	"sigs.k8s.io/controller-runtime/pkg/client"

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
	revision := incarnation.revision
	if revision == nil {
		return nil
	}
	label := fmt.Sprintf("%s%s", picchu.LabelTargetDeletablePrefix, incarnation.targetName())
	revision.Labels[label] = "true"
	return cli.Update(ctx, revision)
}

func markGarbage(ctx context.Context, log logr.Logger, cli client.Client, incarnations []*Incarnation) error {
	revisions := make([]garbagecollector.Revision, 0, len(incarnations))
	for i := range incarnations {
		revisions = append(revisions, incarnations[i])
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
