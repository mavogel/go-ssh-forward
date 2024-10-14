package forward

import (
	"strings"
	"testing"
)

func TestConfigNotSet(t *testing.T) {
	t.Parallel()
	_, _, err := NewForward(nil)
	if err == nil {
		t.Errorf("Expected an error for a nil Config but got none")
	}
	checkErrorContains(t, err, "Config")
}
func TestConfigTooManyJumpHosts(t *testing.T) {
	t.Parallel()
	Config := &Config{
		JumpHostConfigs: []*SSHConfig{
			{
				Address:        "10.0.0.1:22",
				User:           "jumpuser1",
				PrivateKeyFile: "./testing/rsa_keys/id_rsa_jump_host1",
				Password:       "",
			},
			{
				Address:        "10.0.0.2:22",
				User:           "jumpuser2",
				PrivateKeyFile: "./testing/rsa_keys/id_rsa_jump_host2",
				Password:       "",
			},
		},
		EndHostConfig: &SSHConfig{
			Address:        "20.0.0.1:22",
			User:           "endhostuser",
			PrivateKeyFile: "./testing/rsa_keys/id_rsa_end_host",
			Password:       "",
		},
		LocalAddress:  "localhost:2376",
		RemoteAddress: "localhost:2376",
	}

	_, _, err := NewForward(Config)
	if err == nil {
		t.Errorf("Expected an error for a too many jump hosts in Config but got none")
	}
	checkErrorContains(t, err, "only 1 jump host")
}
func TestConfigEmptyJumpHosts(t *testing.T) {
	t.Parallel()
	var emptyJumpHosts []*SSHConfig
	Config := &Config{
		JumpHostConfigs: emptyJumpHosts,
		EndHostConfig: &SSHConfig{
			Address:        "20.0.0.1:22",
			User:           "endhostuser",
			PrivateKeyFile: "./testing/rsa_keys/id_rsa_end_host",
			Password:       "",
		},
		LocalAddress:  "localhost:2376",
		RemoteAddress: "localhost:2376",
	}

	_, _, err := NewForward(Config)
	if err == nil {
		t.Errorf("Expected an error for a end host not reachable with no jump host but got none")
	}
	checkErrorContains(t, err, "ssh.Dial directly to end host failed")
}

func TestConfigNoJumpHostsSet(t *testing.T) {
	t.Parallel()
	Config := &Config{
		EndHostConfig: &SSHConfig{
			Address:        "20.0.0.1:22",
			User:           "endhostuser",
			PrivateKeyFile: "./testing/rsa_keys/id_rsa_end_host",
			Password:       "",
		},
		LocalAddress:  "localhost:2376",
		RemoteAddress: "localhost:2376",
	}

	_, _, err := NewForward(Config)
	if err == nil {
		t.Errorf("Expected an error for a end host not reachable with no jump host but got none")
	}
	checkErrorContains(t, err, "ssh.Dial directly to end host failed")
}
func TestConfigInvalidJumpHostSSHConfig(t *testing.T) {
	t.Parallel()

	Config := createConfig("10.0.0.1:22", "", "./testing/rsa_keys/id_rsa_jump_host1", "")
	_, _, err := NewForward(Config)
	checkErrorContains(t, err, "user cannot be empty")

	Config = createConfig("", "jumpuser", "./testing/rsa_keys/id_rsa_jump_host1", "")
	_, _, err = NewForward(Config)
	checkErrorContains(t, err, "address cannot be empty")

	Config = createConfig("10.0.0.1:22", "jumpuser", "", "")
	_, _, err = NewForward(Config)
	checkErrorContains(t, err, "either PrivateKeyFile or Password")

	Config = createConfig("10.0.0.1:22", "jumpuser", "./testing/rsa_keys/does_not_exist", "")
	_, _, err = NewForward(Config)
	checkErrorContains(t, err, "failed to parse jump host config")

	Config = createConfigWithAddresses("10.0.0.1:22", "jumpuser", "./testing/rsa_keys/id_rsa_jump_host1", "", "localhost:2376", "")
	_, _, err = NewForward(Config)
	checkErrorContains(t, err, "localAddress and RemoteAddress have to be set")

	Config = createConfigWithAddresses("10.0.0.1:22", "jumpuser", "", "mypwd", "", "localhost:2376")
	_, _, err = NewForward(Config)
	checkErrorContains(t, err, "localAddress and RemoteAddress have to be set")
}

func TestForwardUnreachableJumpHost(t *testing.T) {
	Config := createConfig("10.0.0.1:22", "jumpuser", "", "jumpuserpassword")
	_, _, err := NewForward(Config)
	checkErrorContains(t, err, "failed to build SSH client")
	checkErrorContains(t, err, "ssh.Dial to jump host failed")
}

func TestForwardUnreachableEndHostWithoutJump(t *testing.T) {
	Config := createForwardWithoutJump("localhost:2376", "localhost:2376")
	_, _, err := NewForward(Config)
	checkErrorContains(t, err, "failed to build SSH client")
	checkErrorContains(t, err, "ssh.Dial directly to end host failed")
}

// ////////////
// Helpers
// ////////////
func checkErrorContains(t *testing.T, err error, errorMsgtoContain string) {
	if !strings.Contains(err.Error(), errorMsgtoContain) {
		t.Errorf("Expected error to contain \n'%s' but got \n'%s'", errorMsgtoContain, err.Error())
	}
}

func createConfig(jumpHostAddress, jumpHostUser, jumpHostPrivateKeyFile, jumpHostPassword string) *Config {
	return createConfigBase(jumpHostAddress, jumpHostUser, jumpHostPrivateKeyFile, jumpHostPassword, "localhost:2376", "localhost:2376")
}

func createConfigWithAddresses(jumpHostAddress, jumpHostUser, jumpHostPrivateKeyFile, jumpHostPassword, localAddress, remoteAddress string) *Config { //nolint: lll
	return createConfigBase(jumpHostAddress, jumpHostUser, jumpHostPrivateKeyFile, jumpHostPassword, localAddress, remoteAddress)
}

func createConfigBase(jumpHostAddress, jumpHostUser, jumpHostPrivateKeyFile, jumpHostPassword, localAddress, remoteAddress string) *Config {
	return &Config{
		JumpHostConfigs: []*SSHConfig{
			{
				Address:        jumpHostAddress,
				User:           jumpHostUser,
				PrivateKeyFile: jumpHostPrivateKeyFile,
				Password:       jumpHostPassword,
			},
		},
		EndHostConfig: &SSHConfig{
			Address:        "20.0.0.1:22",
			User:           "endhostuser",
			PrivateKeyFile: "./testing/rsa_keys/id_rsa_end_host",
			Password:       "",
		},
		LocalAddress:  localAddress,
		RemoteAddress: remoteAddress,
	}
}

func createForwardWithoutJump(localAddress, remoteAddress string) *Config {
	return &Config{
		JumpHostConfigs: []*SSHConfig{},
		EndHostConfig: &SSHConfig{
			Address:        "20.0.0.1:22",
			User:           "endhostuser",
			PrivateKeyFile: "",
			Password:       "endhostuserpass",
		},
		LocalAddress:  localAddress,
		RemoteAddress: remoteAddress,
	}
}
