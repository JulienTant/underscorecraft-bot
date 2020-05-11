package docker

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	dclient "github.com/docker/docker/client"
)

type Client struct {
	d *dclient.Client
}

type Container struct {
	*types.Container
}

type Attached struct {
	*types.HijackedResponse

	handler map[string]func(s string) error
}

func (a *Attached) SendString(msg string) (err error) {
	_, err = a.Conn.Write([]byte{'\n'})
	if err != nil {
		return fmt.Errorf("first n: %w", err)
	}

	_, err = a.Conn.Write([]byte(msg))
	if err != nil {
		return fmt.Errorf("send msg: %w", err)
	}

	_, err = a.Conn.Write([]byte{'\n'})
	if err != nil {
		return fmt.Errorf("second n: %w", err)
	}

	return nil
}

func (a *Attached) OnNewMessage(name string, f func(s string) error) {
	a.handler[name] = f
}

func (a *Attached) Listen(ctx context.Context, inactivityDuration time.Duration, isAliveFn func() bool) error {
	lastThing := time.Now()
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			_ = a.Conn.SetDeadline(time.Now().Add(5 * time.Second))

			if time.Now().After(lastThing.Add(inactivityDuration)) {
				return fmt.Errorf("no activity for %s, restart", inactivityDuration)
			}

			s, err := a.Reader.ReadString('\n')
			if len(s) == 0 || err != nil && err != io.EOF {
				if !isAliveFn() {
					return errors.New("container is not alive..")
				}
				continue
			}

			for i := range a.handler {
				err = a.handler[i](s)
				if err != nil {
					break
				}
			}

			lastThing = time.Now()
		}
	}
}

func NewClient() (*Client, error) {
	dockerClient, err := dclient.NewEnvClient()
	if err != nil {
		return nil, fmt.Errorf("creating NewEnvClient: %w", err)
	}

	return &Client{
		d: dockerClient,
	}, nil
}

func (c *Client) InnerClient() *dclient.Client {
	return c.d
}

func (c *Client) GetContainerWithLabel(ctx context.Context, label string) (*Container, error) {
	f := filters.NewArgs()
	f.Add("label", label)
	containers, err := c.d.ContainerList(ctx, types.ContainerListOptions{
		All:     true,
		Filters: f,
	})

	if err != nil {
		return nil, fmt.Errorf("containerList: %w", err)
	}

	if len(containers) != 1 {
		return nil, errors.New("unable to find minecraft container")
	}

	if containers[0].State != "running" {
		return nil, errors.New("container is not running")
	}

	return &Container{&containers[0]}, nil
}

func (c *Client) Attach(ctx context.Context, container *Container) (*Attached, error) {
	attachedContainer, err := c.d.ContainerAttach(ctx, container.ID, types.ContainerAttachOptions{
		Stream: true,
		Stdin:  true,
		Stdout: true,
		Stderr: true,
	})
	if err != nil {
		return nil, fmt.Errorf("attach to container: %w", err)
	}

	return &Attached{
		HijackedResponse: &attachedContainer,
		handler:          map[string]func(s string) error{},
	}, nil
}

func (c *Client) IsContainerAlive(ctx context.Context, container *Container, programStart time.Time) bool {
	cir, _ := c.d.ContainerInspect(ctx, container.ID)
	cstart, _ := time.Parse(time.RFC3339, cir.State.StartedAt)
	if cir.State.Running == false || cstart.After(programStart) {
		return false
	}

	return true
}
