package collector

type device struct {
	Address  string `yaml:"address,omitempty"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
}
