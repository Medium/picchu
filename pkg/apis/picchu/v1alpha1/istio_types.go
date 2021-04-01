package v1alpha1

import (
	"github.com/gogo/protobuf/types"
)

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
