package cmd

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"testing"

	"cdr.dev/coder-cli/coder-sdk"
	"cdr.dev/slog/sloggers/slogtest/assert"
)

var write = flag.Bool("write", false, "write to the golden files")

func Test_resourceManager(t *testing.T) {
	// TODO: cleanup
	verbose = true

	const goldenFile = "resourcemanager_test.golden"
	var buff bytes.Buffer
	data := mockResourceTopEntities()
	tests := []struct {
		header  string
		data    entities
		options resourceTopOptions
	}{
		{
			header: "By User",
			data:   data,
			options: resourceTopOptions{
				group:  "user",
				sortBy: "cpu",
			},
		},
		{
			header: "By Org",
			data:   data,
			options: resourceTopOptions{
				group:  "org",
				sortBy: "cpu",
			},
		},
		{
			header: "By Provider",
			data:   data,
			options: resourceTopOptions{
				group:  "provider",
				sortBy: "cpu",
			},
		},
		{
			header: "Sort By Memory",
			data:   data,
			options: resourceTopOptions{
				group:  "user",
				sortBy: "memory",
			},
		},
	}

	for _, tcase := range tests {
		buff.WriteString(fmt.Sprintf("=== TEST: %s\n", tcase.header))
		err := presentEntites(&buff, tcase.data, tcase.options)
		assert.Success(t, "present entities", err)
	}

	assertGolden(t, goldenFile, buff.Bytes())
}

func assertGolden(t *testing.T, path string, output []byte) {
	if *write {
		err := ioutil.WriteFile(path, output, 0777)
		assert.Success(t, "write file", err)
		return
	}
	goldenContent, err := ioutil.ReadFile(path)
	assert.Success(t, "read golden file", err)
	assert.Equal(t, "golden content matches", string(goldenContent), string(output))
}

func mockResourceTopEntities() entities {
	orgIDs := [...]string{randString(10), randString(10), randString(10)}
	imageIDs := [...]string{randString(10), randString(10), randString(10)}
	providerIDs := [...]string{randString(10), randString(10), randString(10)}
	userIDs := [...]string{randString(10), randString(10), randString(10)}
	envIDs := [...]string{randString(10), randString(10), randString(10), randString(10)}

	return entities{
		providers: []coder.KubernetesProvider{
			{
				ID:   providerIDs[0],
				Name: "mars",
			},
			{
				ID:   providerIDs[1],
				Name: "underground",
			},
		},
		users: []coder.User{
			{
				ID:    userIDs[0],
				Name:  "Random",
				Email: "random@coder.com",
			},
			{
				ID:    userIDs[1],
				Name:  "Second Random",
				Email: "second-random@coder.com",
			},
		},
		orgs: []coder.Organization{
			{
				ID:   orgIDs[0],
				Name: "SpecialOrg",

				//! these should probably be fixed, but for now they are just for the count
				Members: []coder.OrganizationUser{{}, {}},
			},
			{
				ID:   orgIDs[1],
				Name: "NotSoSpecialOrg",

				//! these should probably be fixed, but for now they are just for the count
				Members: []coder.OrganizationUser{{}, {}},
			},
		},
		envs: []coder.Environment{
			{
				ID:             envIDs[0],
				ResourcePoolID: providerIDs[0],
				ImageID:        imageIDs[0],
				OrganizationID: orgIDs[0],
				UserID:         userIDs[0],
				Name:           "dev-env",
				ImageTag:       "20.04",
				CPUCores:       12.2,
				MemoryGB:       64.4,
				LatestStat: coder.EnvironmentStat{
					ContainerStatus: coder.EnvironmentOn,
				},
			},
			{
				ID:             envIDs[1],
				ResourcePoolID: providerIDs[1],
				ImageID:        imageIDs[1],
				OrganizationID: orgIDs[1],
				UserID:         userIDs[1],
				Name:           "another-env",
				ImageTag:       "10.2",
				CPUCores:       4,
				MemoryGB:       16,
				LatestStat: coder.EnvironmentStat{
					ContainerStatus: coder.EnvironmentOn,
				},
			},
			{
				ID:             envIDs[2],
				ResourcePoolID: providerIDs[1],
				ImageID:        imageIDs[1],
				OrganizationID: orgIDs[1],
				UserID:         userIDs[1],
				Name:           "yet-another-env",
				ImageTag:       "10.2",
				CPUCores:       100,
				MemoryGB:       2,
				LatestStat: coder.EnvironmentStat{
					ContainerStatus: coder.EnvironmentOn,
				},
			},
		},
		images: map[string]*coder.Image{
			imageIDs[0]: {
				Repository:     "ubuntu",
				OrganizationID: orgIDs[0],
			},
			imageIDs[1]: {
				Repository:     "archlinux",
				OrganizationID: orgIDs[0],
			},
		},
	}
}
