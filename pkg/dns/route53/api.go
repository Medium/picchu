package route53

import (
	"context"
	"errors"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"
	awsutil "go.medium.engineering/picchu/pkg/aws"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var log = logf.Log.WithName("route53_provider")

type Route53ProviderSyncInput struct {
	ClusterName string
	Hostname    string
	Target      string
	Enabled     bool
}

type Route53Provider struct {
	Client  *route53.Client
	Cluster *v1alpha1.Cluster
}

func New(cluster *v1alpha1.Cluster) *Route53Provider {
	return &Route53Provider{
		Client:  route53.New(awsutil.MustLoadDefaultAWSConfig()),
		Cluster: cluster,
	}
}

// Ensure Route53Host
func (r *Route53Provider) Sync() error {
	for _, dns := range r.Cluster.Spec.DNS {
		if dns.Provider != v1alpha1.Route53Provider {
			continue
		}
		var ingress *v1alpha1.IngressInfo
		switch dns.Ingress {
		case v1alpha1.PrivateIngressName:
			ingress = &r.Cluster.Spec.Ingresses.Private
		case v1alpha1.PublicIngressName:
			ingress = &r.Cluster.Spec.Ingresses.Public
		default:
			continue
		}
		for _, host := range dns.Hosts {
			if err := r.syncRecord(host, ingress); err != nil {
				log.Error(err, "Failed to sync record", "Hostname", host)
				return err
			}
		}
	}
	return nil
}

func (r *Route53Provider) findHostedZone(hostname string) (*route53.HostedZone, error) {
	var bestMatch route53.HostedZone
	input := route53.ListHostedZonesInput{}
	req := r.Client.ListHostedZonesRequest(&input)
	p := route53.NewListHostedZonesPaginator(req)
	for p.Next(context.TODO()) {
		page := p.CurrentPage()
		for _, zone := range page.HostedZones {
			if bestMatch.Name != nil && len(*zone.Name) < len(*bestMatch.Name) {
				continue
			}
			if strings.HasSuffix(hostname, *zone.Name) {
				// Make sure the match is on a subdomain boundry and not
				// something like 'longdomain.com' matching 'domain.com'
				idx := len(hostname) - len(*zone.Name) - 1
				if idx >= 0 && hostname[idx] == '.' {
					bestMatch = zone
				}
			}
		}
	}
	if err := p.Err(); err != nil {
		return nil, err
	}
	if bestMatch.Name == nil {
		e := errors.New("No available HostedZone found")
		log.Error(e, "Error finding HostedZone", "Hostname", hostname)
		return nil, errors.New("No available HostedZone found")
	}
	return &bestMatch, nil
}

func (r *Route53Provider) syncRecord(hostname string, ingress *v1alpha1.IngressInfo) error {
	hostedZone, err := r.findHostedZone(hostname)
	if err != nil {
		return err
	}
	if hostedZone == nil {
		return errors.New("Failed to find hostedZone for hostname")
	}
	action := route53.ChangeActionUpsert
	if !r.Cluster.Spec.Enabled {
		action = route53.ChangeActionDelete
	}
	input := route53.ChangeResourceRecordSetsInput{
		HostedZoneId: hostedZone.Id,
		ChangeBatch: &route53.ChangeBatch{
			Changes: []route53.Change{{
				Action: action,
				ResourceRecordSet: &route53.ResourceRecordSet{
					Name:          &hostname,
					Type:          route53.RRTypeA,
					SetIdentifier: &r.Cluster.Name,
					Weight:        aws.Int64(1),
					AliasTarget: &route53.AliasTarget{
						HostedZoneId:         &ingress.HostedZoneId,
						DNSName:              &ingress.DNSName,
						EvaluateTargetHealth: aws.Bool(false),
					},
				},
			}},
			Comment: aws.String("Created by Picchu"),
		},
	}
	req := r.Client.ChangeResourceRecordSetsRequest(&input)
	_, err = req.Send(context.TODO())
	return err
}

func Sync(cluster *v1alpha1.Cluster) error {
	return New(cluster).Sync()
}

func Delete(cluster *v1alpha1.Cluster) error {
	c := cluster.DeepCopy()
	c.Spec.Enabled = false
	return New(c).Sync()
}
