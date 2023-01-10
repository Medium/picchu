package plan

import (
	"fmt"

	picchuv1alpha1 "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"
)

// Receiver pointer functions for SLOConfig can be found next to relevant syncer
type SLOConfig struct {
	SLO    *picchuv1alpha1.ServiceLevelObjective
	App    string
	Name   string
	Tag    string
	Labels picchuv1alpha1.ServiceLevelObjectiveLabels
}

func (s *SLOConfig) totalQuery() string {
	return fmt.Sprintf("%s:%s:%s", sanitizeName(s.App), sanitizeName(s.Name), "total")
}

func (s *SLOConfig) errorQuery() string {
	return fmt.Sprintf("%s:%s:%s", sanitizeName(s.App), sanitizeName(s.Name), "errors")
}

type SlothSLOConfig struct {
	SLO    *picchuv1alpha1.SlothServiceLevelObjective
	App    string
	Name   string
	Tag    string
	Labels picchuv1alpha1.ServiceLevelObjectiveLabels
}

func (s *SlothSLOConfig) totalQuery() string {
	return fmt.Sprintf("%s:%s:%s", sanitizeName(s.App), sanitizeName(s.Name), "total")
}

func (s *SlothSLOConfig) errorQuery() string {
	return fmt.Sprintf("%s:%s:%s", sanitizeName(s.App), sanitizeName(s.Name), "errors")
}
