package utils

import picchu "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"

func VariantEnabled(releaseManager *picchu.ReleaseManager, variant string) bool {
	for _, v := range releaseManager.Spec.Variants {
		if v.Name == variant {
			return v.Enabled
		}
	}
	return false
}
