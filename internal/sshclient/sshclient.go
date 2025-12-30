package sshclient

import (
	"log/slog"
	"os"

	"golang.org/x/crypto/ssh"
)

func CreateSshClient(addr string, user string, keyPath string, keyPassphrase string) (*ssh.Client, error) {
	// Significant components of this taken from example in docs:
	// https://pkg.go.dev/golang.org/x/crypto@v0.36.0/ssh#example-PublicKeys
	// https://pkg.go.dev/golang.org/x/crypto@v0.36.0/ssh#Dial

	// var hostKey ssh.PublicKey

	key, err := os.ReadFile(keyPath)
	if err != nil {
		slog.Error("unable to read private key", "key-path", keyPath, "err", err)
		return nil, err
	}

	var signer ssh.Signer
	if keyPassphrase != "" {
		slog.Debug("parsing private key with provided passphrase", "key-path", keyPath)
		signer, err = ssh.ParsePrivateKeyWithPassphrase(key, []byte(keyPassphrase))
	} else {
		signer, err = ssh.ParsePrivateKey(key)
	}

	if err != nil {
		slog.Error("unable to parse private key - ensure the correct passphrase is provided", "key-path", keyPath, "err", err)
		return nil, err
	}

	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		// HostKeyCallback: ssh.FixedHostKey(hostKey),
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	slog.Debug("dialing ssh server", "addr", addr, "config", config)

	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		slog.Error("unable to connect to remove server", "addr", addr, "config", config, "err", err)
		return nil, err
	}

	slog.Debug("successfully dialed ssh server", "addr", addr, "config", config)
	return client, nil
}
