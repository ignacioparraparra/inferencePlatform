package PodControlPlane

import (
	"context"
	"fmt"
	"inference-platform/LoadBalancer"
	"inference-platform/Logger"
	"log"
	"os"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/flags"
	"github.com/docker/compose/v5/pkg/api"
	"github.com/docker/compose/v5/pkg/compose"
)

var logger *log.Logger

type DockerLogConsumer struct {
	Logger *log.Logger
}

func (d *DockerLogConsumer) Log(container, message string) {
	d.Logger.Printf("[%s] %s", container, message)
}

func (d *DockerLogConsumer) Err(container, message string) {
	d.Logger.Printf("[ERROR] [%s] %s", container, message)
}

func (d *DockerLogConsumer) Status(container, message string) {
	d.Logger.Printf("[STATUS] [%s] %s", container, message)
}

var _ api.LogConsumer = (*DockerLogConsumer)(nil)

type PodController struct {
	addrs    []string
	AddrsSet chan struct{}
}

func NewPodController() *PodController {
	return &PodController{
		addrs:    []string{},
		AddrsSet: make(chan struct{}),
	}
}

func (pc *PodController) StartDockerClient() error {
	ctx := context.Background()

	var logFile *os.File
	var err error

	logger, logFile, err = Logger.CreateLogger("controlPlane.log")

	if err != nil {
		return err
	}

	defer logFile.Close()

	dockerCLI, err := command.NewDockerCli()
	if err != nil {
		logger.Fatalf("Failed to create docker CLI: %v", err)
	}

	err = dockerCLI.Initialize(&flags.ClientOptions{})
	if err != nil {
		logger.Fatalf("Failed to initialize docker CLI: %v", err)
	}

	// Create a new Compose service instance
	service, err := compose.NewComposeService(dockerCLI)
	if err != nil {
		logger.Fatalf("Failed to create compose service: %v", err)
	}

	// Load the Compose project from a compose file
	project, err := service.LoadProject(ctx, api.ProjectLoadOptions{
		ProjectName: "pod-control-plane",
		ConfigPaths: []string{"./PodControlPlane/compose.yaml"},
	})
	if err != nil {
		logger.Fatalf("Failed to load project: %v", err)
	}

	err = service.Scale(ctx, project, api.ScaleOptions{
		Replicas: map[string]int{
			"vllm": 2,
		},
	})

	if err != nil {
		logger.Printf("Failed to scale vLLM: %v", err)
		return err
	}

	containers, err := service.Ps(ctx, project.Name, api.PsOptions{})

	for _, c := range containers {
		if c.Status != "running" {
			continue
		}

		for _, port := range c.Publishers {
			if port.TargetPort == 8000 &&
				port.PublishedPort != 0 &&
				port.Protocol == "tcp" {

				url := fmt.Sprintf(
					"http://127.0.0.1:%d",
					port.PublishedPort,
				)

				pc.addrs = append(pc.addrs, url)
			}
		}
	}

	close(pc.AddrsSet)

	// // Start the services defined in the Compose file
	// err = service.Start(ctx, project.Name, api.StartOptions{})

	// if err != nil {
	// 	logger.Fatalf("Failed to start services: %v", err)
	// }

	logger.Printf("Successfully scaled project: %s", project.Name)

	go service.Events(ctx, project.Name, api.EventsOptions{
		Consumer: func(event api.Event) error {
			logger.Println(event)
			return nil
		},
	})

	consumer := &DockerLogConsumer{
		Logger: logger,
	}

	service.Logs(ctx, project.Name, consumer, api.LogOptions{Follow: true})

	return nil
}

func (pc *PodController) UpdateRegistry(lb *LoadBalancer.Registry) {
	for _, addr := range pc.addrs {
		lb.AddAddress(addr)
		logger.Printf("Added %s to registry\n", addr)
	}
}
