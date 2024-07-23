package cfg

type Root struct {
	RegistryHostname string
	RegistryUser     string
	RegistryPassword string
	SkipTLSVerify    bool
	TLSEnabled       bool
	LogInJSON        bool
}
