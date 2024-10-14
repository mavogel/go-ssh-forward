package forward

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"time"

	"golang.org/x/crypto/ssh"
)

var ( //nolint: gochecknoglobals
	readDeadline                    = 30
	writeDeadline                   = 30
	connectionTimeout               = 8 * time.Second
	ErrDialFromJumpToJumphostFailed = fmt.Errorf("ssh.Dial from jump host to next jump failed after timeout")
)

// SSHConfig for a cleaner version.
type SSHConfig struct {
	Address        string
	User           string
	PrivateKeyFile string
	Password       string
}

// Config the configuration for the forward.
type Config struct {
	JumpHostConfigs []*SSHConfig
	EndHostConfig   *SSHConfig
	LocalAddress    string
	RemoteAddress   string
}

// Forward wraps the forward.
type Forward struct {
	quit          chan bool
	config        *Config
	forwardErrors chan error
}

// NewForward creates new forward with optional jump hosts.
func NewForward(config *Config) (*Forward, chan error, error) {
	// bootstrap
	if err := checkConfig(config); err != nil {
		return nil, nil, err
	}

	forwardErrors := make(chan error)
	t := &Forward{
		quit:          make(chan bool),
		config:        config,
		forwardErrors: forwardErrors,
	}

	sshClient, localListener, err := t.bootstrap()
	if err != nil {
		return nil, nil, err
	}

	// run the forward
	go t.run(sshClient, localListener)

	return t, forwardErrors, nil
}

func (f *Forward) run(sshClient *ssh.Client, localListener net.Listener) { //nolint: cyclop
	defer func() {
		localListener.Close()
		sshClient.Close()
	}()

	jumpCount := 1

	for {
		//nolint: godox
		endHostConn, err := sshClient.Dial("tcp", f.config.RemoteAddress) // TODO timeout here?
		if err != nil {
			f.forwardErrors <- fmt.Errorf("failed to connect on end host to docker daemon at '%s': %w", f.config.RemoteAddress, err)
			return //nolint: nlreturn
		}

		if err = endHostConn.SetReadDeadline(time.Now().Add(time.Duration(readDeadline) * time.Second)); err != nil {
			f.forwardErrors <- fmt.Errorf("failed to set read deadline on end host connection: %w", err)
		}

		if err = endHostConn.SetWriteDeadline(time.Now().Add(time.Duration(writeDeadline) * time.Second)); err != nil {
			f.forwardErrors <- fmt.Errorf("failed to set write deadline on end host connection: %w", err)
		}
		defer endHostConn.Close() //nolint: gocritic

		localConn, err := f.buildLocalConnection(localListener)
		if err != nil {
			f.forwardErrors <- fmt.Errorf("failed to build the local connection: %w", err)
			return //nolint: nlreturn
		}

		if err = localConn.SetReadDeadline(time.Now().Add(time.Duration(readDeadline) * time.Second)); err != nil {
			f.forwardErrors <- fmt.Errorf("failed to set read deadline on local connection: %w", err)
		}

		if err = localConn.SetWriteDeadline(time.Now().Add(time.Duration(writeDeadline) * time.Second)); err != nil {
			f.forwardErrors <- fmt.Errorf("failed to set write deadline on local connection: %w", err)
		}
		defer localConn.Close() //nolint: gocritic

		if err != nil {
			select {
			case <-f.quit:
				return
			default:
				continue
			}
		}

		f.handleForward(localConn, endHostConn)
		jumpCount++
	}
}

// Stop stops the forward.
func (f *Forward) Stop() {
	close(f.quit)
	close(f.forwardErrors)
}

// Initial checking start

// convertToSSHConfig converts the given ssh config into
// and *ssh.ClienConfig while preferring privateKeyFiles.
func convertToSSHConfig(toConvert *SSHConfig) (*ssh.ClientConfig, error) {
	config := &ssh.ClientConfig{
		User:            toConvert.User,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), //nolint: gosec
		Timeout:         connectionTimeout,
	}

	if toConvert.PrivateKeyFile != "" {
		publicKey, err := publicKeyFile(toConvert.PrivateKeyFile)
		if err != nil {
			return nil, err
		}

		config.Auth = []ssh.AuthMethod{publicKey}
	} else {
		config.Auth = []ssh.AuthMethod{ssh.Password(toConvert.Password)}
	}

	return config, nil
}

// publicKeyFile helper to read the key files.
func publicKeyFile(file string) (ssh.AuthMethod, error) {
	buffer, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file '%s': %w", file, err)
	}

	key, err := ssh.ParsePrivateKey(buffer)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key from file '%s': %w", file, err)
	}

	return ssh.PublicKeys(key), nil
}

// Initial checking end

// Bootstrap connection start.
func (f *Forward) bootstrap() (*ssh.Client, net.Listener, error) {
	sshClient, err := f.buildSSHClient()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to build SSH client: %w", err)
	}

	localListener, err := net.Listen("tcp", f.config.LocalAddress)
	if err != nil {
		return nil, nil, fmt.Errorf("net.Listen failed: %w", err)
	}

	return sshClient, localListener, nil
}

// buildSSHClient builds the *ssh.Client connection via the jump
// host to the end host. ATM only one jump host is supported.
func (f *Forward) buildSSHClient() (*ssh.Client, error) {
	endHostConfig, err := convertToSSHConfig(f.config.EndHostConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to parse end host config: %w", err)
	}

	if len(f.config.JumpHostConfigs) > 0 {
		jumpHostConfig, err := convertToSSHConfig(f.config.JumpHostConfigs[0])
		if err != nil {
			return nil, fmt.Errorf("failed to parse jump host config: %w", err)
		}

		jumpHostClient, err := ssh.Dial("tcp", f.config.JumpHostConfigs[0].Address, jumpHostConfig)
		if err != nil {
			return nil, fmt.Errorf("ssh.Dial to jump host failed: %w", err)
		}

		jumpHostConn, err := f.dialNextJump(jumpHostClient, f.config.EndHostConfig.Address)
		if err != nil {
			return nil, fmt.Errorf("ssh.Dial from jump to jump host failed: %w", err)
		}

		ncc, chans, reqs, err := ssh.NewClientConn(jumpHostConn, f.config.EndHostConfig.Address, endHostConfig)
		if err != nil {
			jumpHostConn.Close()

			return nil, fmt.Errorf("failed to create ssh client to end host: %w", err)
		}

		return ssh.NewClient(ncc, chans, reqs), nil
	}

	endHostClient, err := ssh.Dial("tcp", f.config.EndHostConfig.Address, endHostConfig)
	if err != nil {
		return nil, fmt.Errorf("ssh.Dial directly to end host failed: %w", err)
	}

	return endHostClient, nil
}

// dialNextJump dials the next jump host in the chain.
func (f *Forward) dialNextJump(jumpHostClient *ssh.Client, nextJumpAddress string) (net.Conn, error) {
	// NOTE: no timeout param in ssh.Dial: https://github.com/golang/go/issues/20288
	// so we implement it by hand
	var jumpHostConn net.Conn

	connChan := make(chan net.Conn)

	go func() {
		jumpHostConn, err := jumpHostClient.Dial("tcp", nextJumpAddress)
		if err != nil {
			f.forwardErrors <- fmt.Errorf("ssh.Dial from jump host to end server failed: %w", err)
			return //nolint: nlreturn
		}
		connChan <- jumpHostConn
	}()

	select {
	case jumpHostConnSel := <-connChan:
		jumpHostConn = jumpHostConnSel
	case <-time.After(connectionTimeout):
		return nil, ErrDialFromJumpToJumphostFailed
	}

	return jumpHostConn, nil
}

// Bootstrap connection end

// Forward start
// buildLocalConnection builds a connection to the locallistener.
func (f *Forward) buildLocalConnection(localListener net.Listener) (net.Conn, error) {
	localConn, err := localListener.Accept()
	if err != nil {
		return nil, fmt.Errorf("failed to listen to local address: %w", err)
	}

	return localConn, nil
}

// handleForward handles the copy in both direction of the forward tunnel.
func (f *Forward) handleForward(localConn, sshConn net.Conn) {
	go func() {
		_, err := io.Copy(sshConn, localConn) // ssh <- local
		if err != nil {
			log.Fatalf("io.Copy from localAddress -> remoteAddress failed: %v", err)
		}
	}()

	go func() {
		_, err := io.Copy(localConn, sshConn) // local <- ssh
		if err != nil {
			log.Fatalf("io.Copy from remoteAddress -> localAddress failed: %v", err)
		}
	}()
}

// Forward end
