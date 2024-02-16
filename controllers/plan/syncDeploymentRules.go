package plan

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	picchuv1alpha1 "go.medium.engineering/picchu/api/v1alpha1"
	"go.medium.engineering/picchu/plan"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	DeploymentAppLabel = "app"
	DeploymentTagLabel = "tag"
	// should be labeled as an slo - not canary
	DeploymentSLOLabel = "slo"
	// CanaryLabel            = "canary"
	DeploymentChannelLabel = "channel"

	DeploymentSummaryAnnotation = "summary"
	DeploymentMessageAnnotation = "message"

	// RuleTypeDeployment is the value of the LabelRuleType label for deployment rules.
	RuleTypeDeployment = "deployment"
)

type SyncDeploymentRules struct {
	App                         string
	Namespace                   string
	Tag                         string
	Labels                      map[string]string
	ServiceLevelObjectiveLabels picchuv1alpha1.ServiceLevelObjectiveLabels
	ServiceLevelObjectives      []*picchuv1alpha1.SlothServiceLevelObjective
}

func (p *SyncDeploymentRules) Apply(ctx context.Context, cli client.Client, cluster *picchuv1alpha1.Cluster, log logr.Logger) error {
	prometheusRules, err := p.prometheusRules(log)
	if err != nil {
		return err
	}

	if len(prometheusRules.Items) > 0 {
		for _, pr := range prometheusRules.Items {
			if err := plan.CreateOrUpdate(ctx, log, cli, pr); err != nil {
				return err
			}
		}
	}

	return nil
}

func (p *SyncDeploymentRules) prometheusRules(log logr.Logger) (*monitoringv1.PrometheusRuleList, error) {
	prl := &monitoringv1.PrometheusRuleList{}
	prs := []*monitoringv1.PrometheusRule{}

	rule := p.prometheusRule()
	// not looking at slos from <service-name>-slo prometheusRule - creating 2 new slos
	config := SLOConfig{
		App:    p.App,
		Tag:    p.Tag,
		Labels: p.ServiceLevelObjectiveLabels,
	}
	helperRules := config.helperRules(log)
	for _, rg := range helperRules {
		rule.Spec.Groups = append(rule.Spec.Groups, *rg)
	}

	prs = append(prs, rule)
	prl.Items = prs
	return prl, nil
}

func (p *SyncDeploymentRules) prometheusRule() *monitoringv1.PrometheusRule {
	labels := make(map[string]string)

	for k, v := range p.Labels {
		labels[k] = v
	}
	// for k, v := range p.ServiceLevelObjectiveLabels.RuleLabels {
	// 	labels[k] = v
	// }

	labels[picchuv1alpha1.LabelRuleType] = RuleTypeDeployment

	return &monitoringv1.PrometheusRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      p.deploymentRuleName(),
			Namespace: p.Namespace,
			Labels:    labels,
		},
	}
}

// create CrashLoopBackOff and ImagePullBackOff for new deployment pods
func (s *SLOConfig) helperRules(log logr.Logger) []*monitoringv1.RuleGroup {
	ruleGroups := []*monitoringv1.RuleGroup{}

	crashLoopLabels := s.deploymentRuleLabels()
	for k, v := range s.Labels.AlertLabels {
		crashLoopLabels[k] = v
	}

	crashLoopRuleGroup := &monitoringv1.RuleGroup{
		Name: s.crashLoopAlertName(),
		Rules: []monitoringv1.Rule{
			{
				Alert:       s.crashLoopAlertName(),
				For:         monitoringv1.Duration("1m"),
				Expr:        intstr.FromString(s.crashLoopQuery(log)),
				Labels:      crashLoopLabels,
				Annotations: s.crashLoopRuleAnnotations(log),
			},
		},
	}

	ruleGroups = append(ruleGroups, crashLoopRuleGroup)

	imagePullBackOffLabels := s.deploymentRuleLabels()
	for k, v := range s.Labels.AlertLabels {
		imagePullBackOffLabels[k] = v
	}

	imagePullBackOffRuleGroup := &monitoringv1.RuleGroup{
		Name: s.imagePullBackOffAlertName(),
		Rules: []monitoringv1.Rule{
			{
				Alert:       s.imagePullBackOffAlertName(),
				For:         monitoringv1.Duration("1m"),
				Expr:        intstr.FromString(s.imagePullBackOffQuery(log)),
				Labels:      imagePullBackOffLabels,
				Annotations: s.imagePullBackOffAnnotations(log),
			},
		},
	}

	ruleGroups = append(ruleGroups, imagePullBackOffRuleGroup)

	return ruleGroups
}

// this needs to be included in the slos
func (s *SLOConfig) deploymentRuleLabels() map[string]string {
	return map[string]string{
		DeploymentAppLabel: s.App,
		DeploymentTagLabel: s.Tag,
		// CanaryLabel:        "true", NO CANARY! general slo
		DeploymentSLOLabel:     "true",
		DeploymentChannelLabel: "#eng-releases",
	}
}

func (s *SLOConfig) crashLoopRuleAnnotations(log logr.Logger) map[string]string {
	return map[string]string{
		DeploymentSummaryAnnotation: fmt.Sprintf("%s - Deployment is failing CrashLoopBackOff SLO - there is at least one pod in state `CrashLoopBackOff`", s.App),
		DeploymentMessageAnnotation: "There is at least one pod in state `CrashLoopBackOff`",
	}
}

func (s *SLOConfig) imagePullBackOffAnnotations(log logr.Logger) map[string]string {
	return map[string]string{
		DeploymentSummaryAnnotation: fmt.Sprintf("%s - Deployment is failing ImagePullBackOff SLO - there is at least one pod in state `ImagePullBackOff`", s.App),
		DeploymentMessageAnnotation: "There is at least one pod in state `ImagePullBackOff`",
	}
}

func (p *SyncDeploymentRules) deploymentRuleName() string {
	return fmt.Sprintf("%s-deployment-%s", p.App, p.Tag)
}

func (s *SLOConfig) crashLoopAlertName() string {
	// per app not soo
	name := strings.Replace(s.App, "-", "_", -1)
	return fmt.Sprintf("%s_deployment_crashloop", name)
}

func (s *SLOConfig) imagePullBackOffAlertName() string {
	name := strings.Replace(s.App, "-", "_", -1)
	return fmt.Sprintf("%s_deployment_imagepullbackoff", name)
}

func (s *SLOConfig) crashLoopQuery(log logr.Logger) string {
	// pod=~"main-20240109-181554-cba9e8cbbf-.*"
	return fmt.Sprintf("sum by (reason) (kube_pod_container_status_waiting_reason{reason=\"CrashLoopBackOff\", container=\"%s\", pod=~\"%s-.*\"}) > 0",
		s.App, s.Tag,
	)
}

func (s *SLOConfig) imagePullBackOffQuery(log logr.Logger) string {
	return fmt.Sprintf("sum by (reason) (kube_pod_container_status_waiting_reason{reason=\"ImagePullBackOff\", container=\"%s\", pod=~\"%s-.*\"}) > 0",
		s.App, s.Tag,
	)
}
