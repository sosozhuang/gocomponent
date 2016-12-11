package docker

import (
	"fmt"
	"github.com/fsouza/go-dockerclient"
	"bytes"
	"log"
	"github.com/sosozhuang/gocomponent/errors"
)

type Image struct {
	Registry, Repo, Name, Tag string
}

func Client(hostname string, port uint16) (*docker.Client, error){
	endpoint := fmt.Sprintf("tcp://%s:%d", hostname, port)
	return docker.NewClient(endpoint)
}

func BuildAndPush(client *docker.Client, image *Image, contextDir string) error {
	if err := BuildFromDockerfile(client, image, contextDir); err != nil {
		return err
	}
	if err := PushToRegistry(client, image); err != nil {
		return err
	}
	return nil
}

func BuildFromDockerfile(client *docker.Client, image *Image, contextDir string) error {
	var buf bytes.Buffer
	imageName := fmt.Sprintf("%s/%s/%s:%s", image.Registry, image.Repo, image.Name, image.Tag)
	labels := make(map[string]string)
	labels["name"] = imageName
	opts := docker.BuildImageOptions{
		Name:           imageName,
		SuppressOutput: true,
		OutputStream:   &buf,
		RmTmpContainer: true,
		ContextDir:     contextDir,
		Labels: labels,
	}

	if err := client.BuildImage(opts); err != nil {
		return errors.Errorf("Error building image: %s", err)
	}
	return nil
}

func PushToRegistry(client *docker.Client, image *Image) error {
	opts := docker.PushImageOptions{
		Name:     fmt.Sprintf("%s/%s/%s", image.Registry, image.Repo, image.Name),
		Tag:      image.Tag,
		Registry: image.Registry,
	}
	if err := client.PushImage(opts, docker.AuthConfiguration{}); err != nil {
		return errors.Errorf("Error pushing image to registry: %s", err)
	}
	return nil
}

func Run(client *docker.Client, name, image string, env []string) (*docker.Container, error) {
	opts := docker.CreateContainerOptions{
		Name: name,
		Config: &docker.Config{
			Image: image,
			Env: env,
		},
		HostConfig: &docker.HostConfig{
			Privileged: true,
		},
	}
	container, err := Create(client, opts)
	if err != nil {
		return container, err
	}
	err = Start(client, container.ID)
	if err != nil {
		return container, err
	}
	container, err = Inspect(client, container.ID)
	if err != nil {
		return container, err
	}
	if !container.State.Running {
		return container, errors.Errorf("Container %s is not running", container.ID)
	}
	return container, nil
}

func Create(client *docker.Client, opts docker.CreateContainerOptions) (*docker.Container, error) {
	container, err := client.CreateContainer(opts)
	if err != nil {
		return nil, errors.Errorf("Error creating container: %s" ,err)
	}
	log.Printf("Container created, name: %s, id: %s\n", opts.Name, container.ID)
	return container, nil
}

func Start(client *docker.Client, id string) error {
	err := client.StartContainer(id, &docker.HostConfig{})
	if err != nil {
		return errors.Errorf("Error starting container: %s", err)
	}
	log.Printf("Container started, id: %s\n", id)
	return nil
}

func Inspect(client *docker.Client, id string) (*docker.Container, error){
	container, err := client.InspectContainer(id)
	if err != nil {
		return nil, errors.Errorf("Error inspecting container: %s", err)
	}
	return container, nil
}