package image

import (
	"bytes"
	"fmt"
	"github.com/fsouza/go-dockerclient"
	"os"
)

type DockerImage struct {
	Registry, Repo, Name, Tag string
}
func (image DockerImage) String() string {
	return fmt.Sprintf("%s/%s/%s:%s", image.Registry, image.Repo, image.Name, image.Tag)
}
func (image DockerImage) buildImageName() string {
	return fmt.Sprintf("%s/%s:%s", image.Repo, image.Name, image.Tag)
}

func (image DockerImage) Build(client *docker.Client, contextDir string) {
	BuildFromDockerfile(client, image, contextDir)
	ListImages(client, image)
	PushToRegistry(client, image)
}

//func BuildFsm(client *docker.Client, registry, repo, name, tag, contextDir string) {
//	image := DockerImage{
//		registry, repo, name, tag,
//	}
//	BuildFromDockerfile(client, image, contextDir)
//	ListImages(client, image)
//	PushToRegistry(client, image, registry)
//}

func ListImages(client *docker.Client, image DockerImage) {
	//labels := make(map[string][]string, 1)
	imageName := fmt.Sprintf("%v", image)
	//labels["name"] = []string{imageName}
	imgs, err := client.ListImages(docker.ListImagesOptions{
		All:    false,
		Filter: fmt.Sprintf("%s/%s/%s", image.Registry, image.Repo, image.Name),
		//Filters: labels,
	})
	if err != nil {
		panic(err)
	}
	if len(imgs) <= 0 {
		fmt.Printf("can not find any images: %s\n", imageName)
		os.Exit(1)
	}
	for _, img := range imgs {
		fmt.Println("ID: ", img.ID)
		fmt.Println("RepoTags: ", img.RepoTags)
		fmt.Println("Created: ", img.Created)
		fmt.Println("Size: ", img.Size)
		fmt.Println("VirtualSize: ", img.VirtualSize)
		fmt.Println("ParentId: ", img.ParentID)
		fmt.Println("---------------------")
	}
}

func BuildFromDockerfile(client *docker.Client, image DockerImage, contextDir string) {
	//name := "testimage:v1"
	var buf bytes.Buffer
	imageName := fmt.Sprintf("%v", image)
	labels := make(map[string]string, 1)
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
		fmt.Printf("error building image: %s\n", err)
		os.Exit(1)
	}

}

func PushToRegistry(client *docker.Client, image DockerImage) {
	//client.TagImage(image.buildImageName(), docker.TagImageOptions{
	//	Repo: fmt.Sprintf("%s/%s/%s", image.Registry, image.Repo, image.Name),
	//	Tag:  image.Tag,
	//})
	opts := docker.PushImageOptions{
		Name:     fmt.Sprintf("%s/%s/%s", image.Registry, image.Repo, image.Name),
		Tag:      image.Tag,
		Registry: image.Registry,
	}
	if err := client.PushImage(opts, docker.AuthConfiguration{}); err != nil {
		fmt.Printf("error pushing image to registry: %s\n", err)
		os.Exit(1)
	}
}
