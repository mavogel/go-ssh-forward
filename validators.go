package forward

import (
	"errors"
)

var (
	ErrJumpHostsNotSupportd           = errors.New("only 1 jump host is supported atm")             //nolint: revive
	ErrLocalAndRemoteAddressUnset     = errors.New("localAddress and RemoteAddress have to be set") //nolint: revive
	ErrSSHConfigUnset                 = errors.New("SSHConfig cannot be nil")
	ErrUserEmpty                      = errors.New("user cannot be empty")
	ErrAddressEmpty                   = errors.New("address cannot be empty")
	ErrPrivateKeyFileAndPasswordUnset = errors.New("either PrivateKeyFile or Password has to be set")
	ErrConfigIsNil                    = errors.New("Config cannot be nil")
)

// checkConfig checks the config if it is feasible.
func checkConfig(config *Config) error {
	if config == nil {
		return ErrConfigIsNil
	}

	if len(config.JumpHostConfigs) > 1 {
		return ErrJumpHostsNotSupportd
	}

	for _, jumpConfig := range config.JumpHostConfigs {
		if err := checkSSHConfig(jumpConfig); err != nil {
			return err
		}
	}

	if err := checkSSHConfig(config.EndHostConfig); err != nil {
		return err
	}

	if config.LocalAddress == "" || config.RemoteAddress == "" {
		return ErrLocalAndRemoteAddressUnset
	}

	return nil
}

// checkSSHConfig checks the ssh config for feasibility.
func checkSSHConfig(sshConfig *SSHConfig) error {
	if sshConfig == nil {
		return ErrSSHConfigUnset
	}

	if sshConfig.User == "" {
		return ErrUserEmpty
	}

	if sshConfig.Address == "" {
		return ErrAddressEmpty
	}

	if sshConfig.PrivateKeyFile == "" && sshConfig.Password == "" {
		return ErrPrivateKeyFileAndPasswordUnset
	}

	return nil
}
