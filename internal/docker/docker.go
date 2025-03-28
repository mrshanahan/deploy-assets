package docker

import (
	"strings"
	"time"

	"github.com/pkg/errors"
)

type DockerImageEntry struct {
	Repository string
	ID         string
	CreatedAt  time.Time
}

func ParseDockerImageEntries(output string) ([]*DockerImageEntry, error) {
	var entries []*DockerImageEntry
	lines := strings.Split(strings.TrimSpace(output), "\n")
	for i, l := range lines {
		comps := strings.Split(l, ",")
		if len(comps) != 3 {
			return nil, errors.Errorf("Error in `docker image ls` output line %d (wrong number of CSV components): %s", i+1, l)
		}
		createdAt, err := time.Parse("2006-01-02 15:04:05 -0700 MST", comps[2])
		if err != nil {
			return nil, errors.Errorf("Error in `docker image ls` output line %d (invalid date: %v): %s", i+1, err, l)
		}
		entries = append(entries, &DockerImageEntry{
			Repository: comps[0],
			ID:         comps[1],
			CreatedAt:  createdAt,
		})
	}
	return entries, nil
}
