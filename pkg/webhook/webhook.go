package webhook

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	"go.medium.engineering/picchu/pkg/webhook/revision"
	"io/ioutil"
	admissionregistration "k8s.io/api/admissionregistration/v1beta1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
	"os"
	"path"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"time"
)

const (
	LabelName                = "name"
	ServicePort              = 443
	ServiceName              = "webhook"
	ServicePortName          = "webhook"
	ServiceLabelName         = "picchu-webhook"
	SecretName               = "webhook-cert"
	WebhookConfigurationName = "picchu-policy"
	ValidationWebhookName    = "picchu-validation.medium.engineering"
	MutationWebhookName      = "picchu-mutation.medium.engineering"
	CrtDir                   = "/tmp/k8s-webhook-server/serving-certs"
)

func Register(mgr manager.Manager) {
	revision.Register(mgr)
}

// Init creates all necessary infrastructure to support webhooks. Service, Secret, certs and WebhookConfigurations.
// It is intended to run on the same /tmp volume as the picchu operator which requires the certs to be present.
func Init(cli client.Client, targetPort int32, namespace string, log logr.Logger) {
	log = log.WithName("webhook")
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(10)*time.Second)
	defer cancel()
	service := &core.Service{
		ObjectMeta: meta.ObjectMeta{
			Name:      ServiceName,
			Namespace: namespace,
		},
		Spec: core.ServiceSpec{
			Ports: []core.ServicePort{{
				Name:     ServicePortName,
				Protocol: "TCP",
				Port:     ServicePort,
				TargetPort: intstr.IntOrString{
					Type:   intstr.Int,
					IntVal: targetPort,
				},
			}},
			Selector: map[string]string{
				LabelName: ServiceLabelName,
			},
		},
	}
	spec := service.DeepCopy().Spec
	log.Info("Creating service", "service", service)
	controllerutil.CreateOrUpdate(ctx, cli, service, func() error {
		service.Spec = spec
		return nil
	})

	secret := &core.Secret{}
	cert := &Certificate{}
	err := cli.Get(ctx, client.ObjectKey{
		Namespace: namespace,
		Name:      SecretName,
	}, secret)
	if err != nil && !errors.IsNotFound(err) {
		panic(err)
	}
	if err != nil {
		hostname := fmt.Sprintf("%s.%s.svc", ServiceName, namespace)
		cert = MustGenerateSelfSignedCertificate(hostname)
		secret = &core.Secret{
			ObjectMeta: meta.ObjectMeta{
				Namespace: namespace,
				Name:      SecretName,
			},
			Data: map[string][]byte{
				"ca-tls.crt": cert.CaCrt,
				"ca-tls.key": cert.CaKey,
				"tls.key":    cert.Key,
				"tls.crt":    cert.Crt,
			},
			Type: "Opaque",
		}
		log.Info("Creating secret", "secret", secret)
		panicOnError(cli.Create(ctx, secret))
	} else {
		log.Info("Found secret", "secret", secret)
		cert = &Certificate{
			CaCrt: secret.Data["ca-tls.crt"],
			CaKey: secret.Data["ca-tls.key"],
			Crt:   secret.Data["tls.crt"],
			Key:   secret.Data["tls.key"],
		}
	}

	log.Info("Creating cert files")
	panicOnError(os.MkdirAll(CrtDir, os.FileMode(0755)))
	panicOnError(ioutil.WriteFile(path.Join(CrtDir, "tls.crt"), cert.Crt, os.FileMode(0644)))
	panicOnError(ioutil.WriteFile(path.Join(CrtDir, "tls.key"), cert.Key, os.FileMode(0644)))

	validator := &admissionregistration.ValidatingWebhookConfiguration{
		ObjectMeta: meta.ObjectMeta{
			Name: WebhookConfigurationName,
		},
		Webhooks: []admissionregistration.ValidatingWebhook{{
			Name: ValidationWebhookName,
			ClientConfig: admissionregistration.WebhookClientConfig{
				Service: &admissionregistration.ServiceReference{
					Namespace: namespace,
					Name:      ServiceName,
					Path:      pointer.StringPtr("/validate-revisions"),
					Port:      pointer.Int32Ptr(ServicePort),
				},
				CABundle: cert.CABundle(),
			},
			Rules: []admissionregistration.RuleWithOperations{{
				Operations: []admissionregistration.OperationType{
					admissionregistration.Create,
					admissionregistration.Update,
				},
				Rule: admissionregistration.Rule{
					APIGroups:   []string{"picchu.medium.engineering"},
					APIVersions: []string{"v1alpha1"},
					Resources:   []string{"revisions"},
					Scope:       scopeTypePtr(admissionregistration.AllScopes),
				},
			}},
			FailurePolicy:           failurePolicyTypePtr(admissionregistration.Fail),
			TimeoutSeconds:          pointer.Int32Ptr(1),
			AdmissionReviewVersions: []string{"v1", "v1beta1"},
		}},
	}
	vWebhooks := validator.DeepCopy().Webhooks
	log.Info("Creating validation webhook", "webhook", validator)
	controllerutil.CreateOrUpdate(ctx, cli, validator, func() error {
		validator.Webhooks = vWebhooks
		return nil
	})

	mutator := &admissionregistration.MutatingWebhookConfiguration{
		ObjectMeta: meta.ObjectMeta{
			Name: WebhookConfigurationName,
		},
		Webhooks: []admissionregistration.MutatingWebhook{{
			Name: MutationWebhookName,
			ClientConfig: admissionregistration.WebhookClientConfig{
				Service: &admissionregistration.ServiceReference{
					Namespace: namespace,
					Name:      ServiceName,
					Path:      pointer.StringPtr("/mutate-revisions"),
					Port:      pointer.Int32Ptr(ServicePort),
				},
				CABundle: cert.CABundle(),
			},
			Rules: []admissionregistration.RuleWithOperations{{
				Operations: []admissionregistration.OperationType{
					admissionregistration.Create,
					admissionregistration.Update,
				},
				Rule: admissionregistration.Rule{
					APIGroups:   []string{"picchu.medium.engineering"},
					APIVersions: []string{"v1alpha1"},
					Resources:   []string{"revisions"},
					Scope:       scopeTypePtr(admissionregistration.AllScopes),
				},
			}},
			FailurePolicy:           failurePolicyTypePtr(admissionregistration.Fail),
			TimeoutSeconds:          pointer.Int32Ptr(1),
			AdmissionReviewVersions: []string{"v1", "v1beta1"},
		}},
	}
	mWebhooks := mutator.DeepCopy().Webhooks
	log.Info("Creating mutating webhook", "webhook", mutator)
	panicOnError(controllerutil.CreateOrUpdate(ctx, cli, mutator, func() error {
		mutator.Webhooks = mWebhooks
		return nil
	}))
}

func failurePolicyTypePtr(f admissionregistration.FailurePolicyType) *admissionregistration.FailurePolicyType {
	return &f
}

func scopeTypePtr(s admissionregistration.ScopeType) *admissionregistration.ScopeType {
	return &s
}

func panicOnError(vargs ...interface{}) {
	switch v := vargs[len(vargs)-1].(type) {
	case error:
		if v != nil {
			panic(v)
		}
	}
}
