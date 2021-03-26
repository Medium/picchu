package v1alpha1

import (
	"github.com/gogo/protobuf/types"
)

//
//const (
//	// Range of a Duration in seconds, as specified in
//	// google/protobuf/duration.proto. This is about 10,000 years in seconds.
//	maxSeconds = int64(10000 * 365.25 * 24 * 60 * 60)
//	minSeconds = -maxSeconds
//)
//
//type Duration types.Duration
//
//func validateDuration(d *Duration) error {
//	if d == nil {
//		return errors.New("duration: nil Duration")
//	}
//	if d.Seconds < minSeconds || d.Seconds > maxSeconds {
//		return fmt.Errorf("duration: %#v: seconds out of range", d)
//	}
//	if d.Nanos <= -1e9 || d.Nanos >= 1e9 {
//		return fmt.Errorf("duration: %#v: nanos out of range", d)
//	}
//	// Seconds and Nanos must have the same sign, unless d.Nanos is zero.
//	if (d.Seconds < 0 && d.Nanos > 0) || (d.Seconds > 0 && d.Nanos < 0) {
//		return fmt.Errorf("duration: %#v: seconds and nanos have different signs", d)
//	}
//	return nil
//}
//func DurationFromProto(p *Duration) (time.Duration, error) {
//	if err := validateDuration(p); err != nil {
//		return 0, err
//	}
//	d := time.Duration(p.Seconds) * time.Second
//	if int64(d/time.Second) != p.Seconds {
//		return 0, fmt.Errorf("duration: %#v is out of range for time.Duration", p)
//	}
//	if p.Nanos != 0 {
//		d += time.Duration(p.Nanos) * time.Nanosecond
//		if (d < 0) != (p.Nanos < 0) {
//			return 0, fmt.Errorf("duration: %#v is out of range for time.Duration", p)
//		}
//	}
//	return d, nil
//}

type Duration types.Duration
type BoolValue types.BoolValue
type UInt32Value types.UInt32Value

type TrafficPolicy struct {
	LoadBalancer      *LoadBalancerSettings              `json:"load_balancer,omitempty"`
	ConnectionPool    *ConnectionPoolSettings            `json:"connection_pool,omitempty"`
	OutlierDetection  *OutlierDetection                  `json:"outlier_detection,omitempty"`
	Tls               *ClientTLSSettings                 `json:"tls,omitempty"`
	PortLevelSettings []*TrafficPolicy_PortTrafficPolicy `json:"port_level_settings,omitempty"`
}

type LoadBalancerSettings struct {
	LocalityLbSetting *LocalityLoadBalancerSetting `json:"locality_lb_setting,omitempty"`
}

type LocalityLoadBalancerSetting struct {
	Distribute []*LocalityLoadBalancerSetting_Distribute `json:"distribute,omitempty"`
	Failover   []*LocalityLoadBalancerSetting_Failover   `json:"failover,omitempty"`
	Enabled    *BoolValue                                `json:"enabled,omitempty"`
}

type LocalityLoadBalancerSetting_Distribute struct {
	From string            `json:"from,omitempty"`
	To   map[string]uint32 `json:"to,omitempty"`
}

type LocalityLoadBalancerSetting_Failover struct {
	From string `json:"from,omitempty"`
	To   string `json:"to,omitempty"`
}

type ConnectionPoolSettings struct {
	Tcp  *ConnectionPoolSettings_TCPSettings  `json:"tcp,omitempty"`
	Http *ConnectionPoolSettings_HTTPSettings `json:"http,omitempty"`
}

type ConnectionPoolSettings_TCPSettings struct {
	MaxConnections int32                                            `json:"max_connections,omitempty"`
	ConnectTimeout *Duration                                        `json:"connect_timeout,omitempty"`
	TcpKeepalive   *ConnectionPoolSettings_TCPSettings_TcpKeepalive `json:"tcp_keepalive,omitempty"`
}

type ConnectionPoolSettings_TCPSettings_TcpKeepalive struct {
	Probes   uint32    `json:"probes,omitempty"`
	Time     *Duration `json:"time,omitempty"`
	Interval *Duration `json:"interval,omitempty"`
}

type ConnectionPoolSettings_HTTPSettings struct {
	Http1MaxPendingRequests  int32                                               `json:"http1_max_pending_requests,omitempty"`
	Http2MaxRequests         int32                                               `json:"http2_max_requests,omitempty"`
	MaxRequestsPerConnection int32                                               `json:"max_requests_per_connection,omitempty"`
	MaxRetries               int32                                               `json:"max_retries,omitempty"`
	IdleTimeout              *Duration                                           `json:"idle_timeout,omitempty"`
	H2UpgradePolicy          ConnectionPoolSettings_HTTPSettings_H2UpgradePolicy `json:"h2_upgrade_policy,omitempty"`
	UseClientProtocol        bool                                                `json:"use_client_protocol,omitempty"`
}

type ConnectionPoolSettings_HTTPSettings_H2UpgradePolicy int32

type OutlierDetection struct {
	ConsecutiveErrors        int32        `json:"consecutive_errors,omitempty"`
	ConsecutiveGatewayErrors *UInt32Value `json:"consecutive_gateway_errors,omitempty"`
	Consecutive_5XxErrors    *UInt32Value `json:"consecutive_5xx_errors,omitempty"`
	Interval                 *Duration    `json:"interval,omitempty"`
	BaseEjectionTime         *Duration    `json:"base_ejection_time,omitempty"`
	MaxEjectionPercent       int32        `json:"max_ejection_percent,omitempty"`
	MinHealthPercent         int32        `json:"min_health_percent,omitempty"`
}

type ClientTLSSettings struct {
	Mode              ClientTLSSettings_TLSmode `json:"mode,omitempty"`
	ClientCertificate string                    `json:"client_certificate,omitempty"`
	PrivateKey        string                    `json:"private_key,omitempty"`
	CaCertificates    string                    `json:"ca_certificates,omitempty"`
	CredentialName    string                    `json:"credential_name,omitempty"`
	SubjectAltNames   []string                  `json:"subject_alt_names,omitempty"`
	Sni               string                    `json:"sni,omitempty"`
}

type ClientTLSSettings_TLSmode int32

type TrafficPolicy_PortTrafficPolicy struct {
	Port             *PortSelector           `json:"port,omitempty"`
	LoadBalancer     *LoadBalancerSettings   `json:"load_balancer,omitempty"`
	ConnectionPool   *ConnectionPoolSettings `json:"connection_pool,omitempty"`
	OutlierDetection *OutlierDetection       `json:"outlier_detection,omitempty"`
	Tls              *ClientTLSSettings      `json:"tls,omitempty"`
}

type PortSelector struct {
	Number uint32 `json:"number,omitempty"`
}
