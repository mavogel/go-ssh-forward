[![MIT license](http://img.shields.io/badge/license-MIT-brightgreen.svg)](http://opensource.org/licenses/MIT)
[![PkgGoDev](https://pkg.go.dev/badge/github.com/mavogel/go-ssh-forward)](https://pkg.go.dev/github.com/mavogel/go-ssh-forward)
![Build Status](https://github.com/mavogel/go-ssh-forward/actions/workflows/main.yml/badge.svg)
[![Coverage Status](https://img.shields.io/coveralls/mavogel/go-ssh-forward.svg)](https://coveralls.io/r/mavogel/go-ssh-forward)
[![Go Report Card](https://goreportcard.com/badge/github.com/mavogel/go-ssh-forward)](https://goreportcard.com/report/github.com/mavogel/go-ssh-forward)

# go-ssh-forward: A library for setting up a Forward via SSH in go
## Table of Contents
- [Motivation](#motivation)
- [Usage](#usage)
- [Inspiration](#inspiration)
- [Releasee](#release)
- [License](#license)

## <a name="motivation"></a>Motivation
I wanted it to be possible to establish a tunnel like the following in `go`
```sh
$ ssh -f -L 2376:localhost:2376 \ 
  -o ExitOnForwardFailure=yes \ 
	-o ProxyCommand="ssh -l jumpuser1 -i /Users/abc/.ssh/id_rsa_jump_host1 10.0.0.1 -W %h:%p" \ 
	-o UserKnownHostsFile=/dev/null \ 
	-o StrictHostKeyChecking=no \ 
	-i /Users/abc/.ssh/id_rsa_end_host \ 
	endhostuser@20.0.0.1 \
	sleep 10
```

In this scenario the `end host` is only accessible via the `jump host`
```
    localhost:2376 --(j)--> 10.0.0.1:22 --(e)--> 20.0.0.1:2376 -> 127.0.0.1:2376
       `host A`              `jump host`          `end host          `end host`          
```

## <a name="usage"></a>Usage
```go
package main

import (
	"log"
	"time"

  fw "github.com/mavogel/go-ssh-forward"
)
func main() {
  forwardConfig := &fw.Config{
		JumpHostConfigs: []*fw.SSHConfig{
			&fw.SSHConfig{
				Address:        "10.0.0.1:22",
				User:           "jumpuser1",
				PrivateKeyFile: "/Users/abc/.ssh/id_rsa_jump_host1",
			},
		},
		EndHostConfig: &fw.SSHConfig{
			Address:        "20.0.0.1:22",
			User:           "endhostuser",
			PrivateKeyFile: "/Users/abc/.ssh/id_rsa_end_host",
		},
		LocalAddress:  "localhost:2376",
		RemoteAddress: "localhost:2376",
  }
 
  forward, forwardErrors, bootstrapErr := fw.NewForward(forwardConfig)
  handleForwardErrors(forward)
  defer forward.Stop()
  if bootstrapErr != nil {
		log.Printf("bootstrapErr: %s", bootstrapErr)
		return
  }
  
  // run commands against 127.0.0.1:2376
  // ...

}

func handleForwardErrors(forwardErrors chan error) {
	go func() {
		for {
			select {
			case forwardError := <-forwardErrors:
				log.Printf("forward err: %s", forwardError)
			case <-time.After(3 * time.Second):
				log.Printf("NO forward err...")
			}
		}
	}()
}
```

## <a name="inspiration"></a>Inspiration
- [ssh via bastion](https://stackoverflow.com/questions/35906991/go-x-crypto-ssh-how-to-establish-ssh-connection-to-private-instance-over-a-ba)
- [Go Best Practices](https://talks.golang.org/2013/bestpractices.slide#29) 
- [Shutdown go listeners](http://zhen.org/blog/graceful-shutdown-of-go-net-dot-listeners/)
- [sshego](https://github.com/glycerine/sshego)

## <a name="release"></a>Release
```sh
$ VERSION=vX.Y.Z make release
# EXAMPLE:
$ VERSION=v0.11.3 make release
```

## <a name="license"></a>License
    Copyright (c) 2018 Manuel Vogel
    Source code is open source and released under the MIT license.
