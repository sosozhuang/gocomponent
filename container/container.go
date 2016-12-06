package container

import (
	"fmt"
	"os"
	"github.com/fsouza/go-dockerclient"
)

//func getDockerClient(host string, port uint16) (*docker.Client, error) {
//	endpoint := fmt.Sprintf("tcp://%s:%d", host, port)
//	return docker.NewClient(endpoint)
//}
//
//func getDockerClient1(host DockerHost) (*docker.Client, error) {
//	endpoint := fmt.Sprintf("tcp://%s:%d", host.Hostname, host.Port)
//	return docker.NewClient(endpoint)
//}

func Run(client *docker.Client, name, image string, env []string) {
	//create container
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
	//container, err := client.CreateContainer(opts)
	//if err != nil {
	//	fmt.Printf("error creating container: %s\n", err)
	//	os.Exit(1)
	//}
	container := Create(client, opts)

	//start container
	Start(client, container.ID)

	//inspect container
	Inspect(client, container.ID)
}

func Create(client *docker.Client, opts docker.CreateContainerOptions) (*docker.Container) {
	//client, err := dockerHost.DockerClient()
	//if err != nil {
	//	fmt.Printf("error getting docker client: %s\n", err)
	//	os.Exit(1)
	//}
	//name := "test-create-container"
	//env := []string{"fsm_id=999"}
	container, err := client.CreateContainer(opts)
	if err != nil {
		fmt.Printf("error creating container: %s\n", err)
		os.Exit(1)
	}
	fmt.Printf("container created, name: %s, id: %s\n", opts.Name, container.ID)
	return container
}

func Start(client *docker.Client, id string) {
	err := client.StartContainer(id, &docker.HostConfig{})
	if err != nil {
		fmt.Printf("error starting container: %s\n", err)
		os.Exit(1)
	}
	fmt.Printf("container started, id: %s\n", id)
}

func Inspect(client *docker.Client, id string) {
	container, err := client.InspectContainer(id)
	if err != nil {
		fmt.Printf("error inspecting container: %s\n", err)
		os.Exit(1)
	}
	if !container.State.Running {
		fmt.Printf("container %s is not running\n", id)
		os.Exit(1)
	}
}


