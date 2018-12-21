package iaas

import (
	"fmt"
	"time"

	"github.com/sacloud/libsacloud/sacloud"
)

func (c *client) Servers() ([]sacloud.Server, error) {
	return c.getAPIClient().FindServers()
}

func (c *client) ShutdownServerByID(id int64, shutdownWait time.Duration) error {
	err := c.getAPIClient().ShutdownServer(id, shutdownWait)
	if err != nil {
		return fmt.Errorf("shutdown server %q is failed: %s", id, err)
	}
	return nil
}
