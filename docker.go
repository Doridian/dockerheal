package main

import (
	"context"
	"log"
	"os"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/client"
)

type DockerhealClient struct {
	client *client.Client
}

func NewDockerhealClient() (*DockerhealClient, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}

	return &DockerhealClient{
		client: cli,
	}, nil
}

func (c *DockerhealClient) CheckOnce(ctx context.Context) error {
	containers, err := c.client.ContainerList(ctx, container.ListOptions{})
	if err != nil {
		return err
	}

	for _, ct := range containers {
		containerDetails, err := c.client.ContainerInspect(ctx, ct.ID)
		if err != nil {
			return err
		}

		if containerDetails.State.Health == nil {
			c.reportNoHealth(containerDetails)
			continue
		}

		c.reportHealth(ct.ID, containerDetails.State.Health.Status)
	}

	return nil
}

func (c *DockerhealClient) CheckBackground(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				err := c.CheckOnce(ctx)
				if err != nil {
					log.Printf("Error checking containers: %v", err)
				}
				time.Sleep(30 * time.Second)
			}
		}
	}()
}

func (c *DockerhealClient) Listen(ctx context.Context) (err error) {
	msgChan, errChan := c.client.Events(ctx, events.ListOptions{})

	for {
		select {
		case <-ctx.Done():
			return nil
		case msg := <-msgChan:
			if msg.Type != events.ContainerEventType {
				continue
			}

			actionSplit := strings.Split(string(msg.Action), ":")
			if actionSplit[0] != "health_status" || len(actionSplit) < 2 {
				continue
			}

			healthStatus := strings.TrimSpace(actionSplit[1])

			c.reportHealth(msg.Actor.ID, healthStatus)
		case err = <-errChan:
			return
		}
	}
}

func (c *DockerhealClient) reportHealth(containerID string, healthState string) {
	log.Printf("Container %s reported health %s", containerID, healthState)
	if healthState == "unhealthy" || healthState == "exited" {
		go c.restartContainer(containerID)
	}
}

func (c *DockerhealClient) reportNoHealth(ct container.InspectResponse) {
	log.Printf("Container %s reported status %s", ct.ID, ct.State.Status)

	if ct.State.OOMKilled {
		go c.restartContainer(ct.ID)
		return
	}

	if ct.State.Status == container.StateExited && (ct.State.ExitCode != 0 || ct.HostConfig.RestartPolicy.IsAlways()) {
		go c.restartContainer(ct.ID)
		return
	}
}

func (c *DockerhealClient) restartContainer(containerID string) {
	disableFile := os.Getenv("DISABLE_FILE")
	if disableFile != "" {
		st, err := os.Stat(disableFile)
		if err == nil && st.Size() > 0 {
			log.Printf("Disabled! Skipping restart of container %s", containerID)
			return
		}
	}

	log.Printf("Restarting container %s", containerID)
	err := c.client.ContainerRestart(context.Background(), containerID, container.StopOptions{})
	if err != nil {
		log.Printf("Restarting container %s failed: %v", containerID, err)
		return
	}
	log.Printf("Restarted container %s", containerID)
}
