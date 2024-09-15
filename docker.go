package main

import (
	"context"
	"log"
	"strings"

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

	for _, container := range containers {
		containerDetails, err := c.client.ContainerInspect(ctx, container.ID)
		if err != nil {
			return err
		}

		if containerDetails.State.Dead || containerDetails.State.OOMKilled {
			c.reportHealth(container.ID, "dead")
			continue
		}

		if containerDetails.State.Health == nil {
			continue
		}

		c.reportHealth(container.ID, containerDetails.State.Health.Status)
	}

	return nil
}

func (c *DockerhealClient) Listen(ctx context.Context) error {
	msgChan, errChan := c.client.Events(ctx, events.ListOptions{})

	err := c.CheckOnce(ctx)
	if err != nil {
		return err
	}

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
			return err
		}
	}
}

func (c *DockerhealClient) reportHealth(containerID string, healthState string) {
	log.Printf("Container %s reported health %s", containerID, healthState)
	if healthState == "unhealthy" || healthState == "dead" {
		go c.restartContainer(containerID)
	}
}

func (c *DockerhealClient) restartContainer(containerID string) {
	log.Printf("Restarting container %s", containerID)
	err := c.client.ContainerRestart(context.Background(), containerID, container.StopOptions{})
	if err != nil {
		log.Printf("Restarting container %s failed: %v", containerID, err)
		return
	}
	log.Printf("Restarted container %s", containerID)
}
