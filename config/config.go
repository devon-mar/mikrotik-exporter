package config

import (
	"io"

	yaml "gopkg.in/yaml.v3"
)

type Module struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`

	EnableTLS   bool `yaml:"enable_tls"`
	InsecureTLS bool `yaml:"insecure_tls"`

	Timeout int `yaml:"timeout"`

	Features Features `yaml:"features"`
}

type Features struct {
	BGP       bool `yaml:"bgp,omitempty"`
	Conntrack bool `yaml:"conntrack,omitempty"`
	DHCP      bool `yaml:"dhcp,omitempty"`
	DHCPL     bool `yaml:"dhcpl,omitempty"`
	DHCPv6    bool `yaml:"dhcpv6,omitempty"`
	Firmware  bool `yaml:"firmware,omitempty"`
	Health    bool `yaml:"health,omitempty"`
	Routes    bool `yaml:"routes,omitempty"`
	POE       bool `yaml:"poe,omitempty"`
	Pools     bool `yaml:"pools,omitempty"`
	Optics    bool `yaml:"optics,omitempty"`
	W60G      bool `yaml:"w60g,omitempty"`
	WlanSTA   bool `yaml:"wlansta,omitempty"`
	Capsman   bool `yaml:"capsman,omitempty"`
	WlanIF    bool `yaml:"wlanif,omitempty"`
	Monitor   bool `yaml:"monitor,omitempty"`
	Ipsec     bool `yaml:"ipsec,omitempty"`
	Lte       bool `yaml:"lte,omitempty"`
	Netwatch  bool `yaml:"netwatch,omitempty"`
}

// Config represents the configuration for the exporter
type Config struct {
	Devices  []Device          `yaml:"devices"`
	Features Features          `yaml:"features,omitempty"`
	Modules  map[string]Module `yaml:"modules"`
}

// Device represents a target device
type Device struct {
	Name     string `yaml:"name"`
	Address  string `yaml:"address,omitempty"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Port     string `yaml:"port"`
}

type DnsServer struct {
	Address string `yaml:"address"`
	Port    int    `yaml:"port"`
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
