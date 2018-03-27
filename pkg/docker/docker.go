package docker

import (
	"fmt"
	"os"

	"github.com/docker/docker/client"
	"golang.org/x/crypto/ssh/terminal"
)

// ImageConfig contains all the information needed to interact with the images.
// This is very high level stuff because the purpose of the package is very
// high level.
type ImageConfig struct {
	// BaseName corresponds to the repository name of the image and the name
	// of the container.
	BaseName string
	// BaseImage is the name of the image in repo:tag format.
	BaseImage string
	// Shell is the path to the shell program to use as the ENTRYPOINT.
	Shell string
	// Mount is the directory where a volume should be mounted.
	Mount string
}

// ContainerConfig contains all the information needed to create a container.
type ContainerConfig struct {
	// Name is the container name that will be assigned.
	Name string
	// ImageName is the name of the image to use for the container, in
	// repo:tag format.
	ImageName string
	// Mounts get created as bound volumes in the container.
	Mounts []Mount
	// Env is all the environment variables to set in the container. Every entry
	// in the slice should be of the form KEY=VAL
	Env []string
}

// Mount is directory on the host paired with a volume mount point.
type Mount struct {
	Source      string
	Destination string
}

func (b Mount) String() string {
	return fmt.Sprintf("%v:%v", b.Source, b.Destination)
}

// Client is a Docker client with some extra data attached to it.
//
// It has pointers to stdin, stdout and stderr that need to be initialized so
// only use it by calling `NewClient`.
type Client struct {
	*client.Client

	stdin  *os.File
	stdout *os.File
	stderr *os.File
}

// NewClient returns a `*Client` with stdin, stdout and stderr initialized.
func NewClient() (*Client, error) {
	cli, err := client.NewEnvClient()
	if err != nil {
		return nil, err
	}

	return &Client{
		Client: cli,
		stdin:  os.Stdin,
		stdout: os.Stdout,
		stderr: os.Stderr,
	}, nil
}

// MakeRawTerminal sets the terminal currently pointed to by stdin and stdout
// into a raw terminal. This is necessary for communication with the Docker
// container over the attach socket. If anything goes wrong, it returns an error
// and tries to restore the terminal to its previous state, but it might not
// succeed in doing so depending on what the issue was. If it was successful,
// it returns two callback functions for the caller to restore the terminal
// back to its previous state when ready. The first is for stdout, the second
// is for stderr.
func (c *Client) MakeRawTerminal() (func() error, func() error, error) {
	// This stuff is required to make interactive sessions in the container
	// less buggy. For example, without it, any command typed at the prompt will
	// get repeated out before printing the execution results.
	oldStdout, err := terminal.MakeRaw(int(c.stdout.Fd()))
	if err != nil {
		return nil, nil, err
	}

	restoreStdout := func() error {
		return terminal.Restore(int(c.stdout.Fd()), oldStdout)
	}

	oldStdin, err := terminal.MakeRaw(int(c.stdin.Fd()))
	if err != nil {
		terminal.Restore(int(c.stdout.Fd()), oldStdout)
		return nil, nil, err
	}

	restoreStdin := func() error {
		return terminal.Restore(int(c.stdin.Fd()), oldStdin)
	}

	return restoreStdout, restoreStdin, nil
}
