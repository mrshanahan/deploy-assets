package ssh

import (
	"bytes"
	"log"
	"os"

	"golang.org/x/crypto/ssh"
)

type SSHClient struct {
	Client *ssh.Client
}

func (c *SSHClient) Close() {
	c.Client.Close()
}

func OpenSSHConnection(addr string, user string, keyPath string) (*SSHClient, error) {
	// Significant components of this taken from example in docs:
	// https://pkg.go.dev/golang.org/x/crypto@v0.36.0/ssh#example-PublicKeys
	// https://pkg.go.dev/golang.org/x/crypto@v0.36.0/ssh#Dial

	// var hostKey ssh.PublicKey

	key, err := os.ReadFile(keyPath)
	if err != nil {
		log.Fatalf("unable to read private key: %v", err)
		return nil, err
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		log.Fatalf("unable to parse private key: %v", err)
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

	log.Printf("dialing...")

	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		log.Fatalf("unable to connect to remove server: %v", err)
		return nil, err
	}

	log.Printf("dialed!")
	return &SSHClient{client}, nil
}

func (c *SSHClient) ExecuteCommand(cmd string) (string, error) {
	// Each ClientConn can support multiple interactive sessions,
	// represented by a Session.
	session, err := c.Client.NewSession()
	if err != nil {
		log.Fatal("Failed to create session: ", err)
		return "", err
	}
	defer session.Close()

	// Once a Session is created, you can execute a single command on
	// the remote side using the Run method.
	var b bytes.Buffer
	session.Stdout = &b
	log.Printf("running...")
	if err := session.Run(cmd); err != nil {
		log.Fatal("Failed to run: " + err.Error())
		return "", err
	}

	// fmt.Println(b.String())
	output := b.String()
	return output, nil
}
