package config

import (
	"io"

	yaml "gopkg.in/yaml.v3"
)

type Module struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`

	UsernameFile string `yaml:"username_file"`
	PasswordFile string `yaml:"password_file"`

	EnableTLS   bool `yaml:"enable_tls"`
	InsecureTLS bool `yaml:"insecure_tls"`

	Timeout int `yaml:"timeout"`

	Features Features `yaml:"features"`

	CACert string `yaml:"ca_cert"`
}

type Features struct {
	BGP       bool `yaml:"bgp,omitempty"`
	Conntrack bool `yaml:"conntrack,omitempty"`
	Capsman   bool `yaml:"capsman,omitempty"`
	DHCP      bool `yaml:"dhcp,omitempty"`
	DHCPL     bool `yaml:"dhcpl,omitempty"`
	DHCPv6    bool `yaml:"dhcpv6,omitempty"`
	Firmware  bool `yaml:"firmware,omitempty"`
	Health    bool `yaml:"health,omitempty"`
	Lte       bool `yaml:"lte,omitempty"`
	Interface bool `yaml:"interface,omitempty"`
	Ipsec     bool `yaml:"ipsec,omitempty"`
	Monitor   bool `yaml:"monitor,omitempty"`
	Optics    bool `yaml:"optics,omitempty"`
	POE       bool `yaml:"poe,omitempty"`
	Pools     bool `yaml:"pools,omitempty"`
	Resource  bool `yaml:"resource,omitempty"`
	Routes    bool `yaml:"routes,omitempty"`
	W60G      bool `yaml:"w60g,omitempty"`
	WlanSTA   bool `yaml:"wlansta,omitempty"`
	WlanIF    bool `yaml:"wlanif,omitempty"`
	Netwatch  bool `yaml:"netwatch,omitempty"`
}

// Config represents the configuration for the exporter
type Config struct {
	Modules map[string]Module `yaml:"modules"`
}

// Load reads YAML from reader and unmashals in Config
func Load(r io.Reader) (*Config, error) {
	d := yaml.NewDecoder(r)
	d.KnownFields(true)

	c := &Config{}
	err := d.Decode(c)
	if err != nil {
		return nil, err
	}

	return c, nil
}
