package main

import (
	"flag"
	"log"
	"os"

	"github.com/mrshanahan/deploy-assets/internal/docker"
	"github.com/mrshanahan/deploy-assets/internal/localproc"
	"github.com/mrshanahan/deploy-assets/internal/ssh"
)

func main() {
	var serverParam *string = flag.String("server", "", "remote server to connect to")
	var userParam *string = flag.String("user", "", "user to authenticate with")
	var keyParam *string = flag.String("key-path", "", "path to the private key")
	flag.Parse()

	if *serverParam == "" {
		log.Fatalf("-server param required")
		os.Exit(1)
	}
	if *userParam == "" {
		log.Fatalf("-user param required")
		os.Exit(1)
	}
	if *keyParam == "" {
		log.Fatalf("-key-path param required")
		os.Exit(1)
	}

	client, err := ssh.OpenSSHConnection(*serverParam, *userParam, *keyParam)
	if err != nil {
		log.Fatalf("unable to open connection to %s@%s using key %s: %v", *userParam, *serverParam, *keyParam, err)
		os.Exit(1)
	}
	defer client.Close()

	remoteResult, err := client.ExecuteCommand("sudo docker image ls --format '{{ .Repository }},{{ .ID }},{{ .CreatedAt }}' 'notes-api/*'")
	if err != nil {
		log.Fatalf("unable to execute remote command: %v", err)
		os.Exit(1)
	}
	remoteEntries, err := docker.ParseDockerImageEntries(remoteResult)
	if err != nil {
		log.Fatalf("failed to parse remote `docker image ls` entries: %v", err)
		os.Exit(1)
	}

	stdout, _, err := localproc.ExecuteCommand("docker", "image", "ls", "--format", "{{ .Repository }},{{ .ID }},{{ .CreatedAt }}", "notes-api/*")
	if err != nil {
		log.Fatalf("unable to execute local command: %v", err)
		os.Exit(1)
	}
	localEntries, err := docker.ParseDockerImageEntries(stdout)
	if err != nil {
		log.Fatalf("failed to parse local `docker image ls` entries: %v", err)
		os.Exit(1)
	}

	remoteEntriesMap := make(map[string]*docker.DockerImageEntry)
	for _, e := range remoteEntries {
		remoteEntriesMap[e.Repository] = e
	}

	localEntriesMap := make(map[string]*docker.DockerImageEntry)
	for _, e := range localEntries {
		localEntriesMap[e.Repository] = e
	}

	for k, v := range remoteEntriesMap {
		if e, exists := localEntriesMap[k]; exists && e.ID != v.ID {
			log.Printf("Image mismatch on %s: local=%s, remote=%s\n", k, v.ID, e.ID)
		}
	}
}
