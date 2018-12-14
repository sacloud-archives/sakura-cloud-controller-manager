package iaas

import (
	"fmt"
	"time"

	"github.com/sacloud/libsacloud/sacloud"
)

func (c *client) Servers() ([]sacloud.Server, error) {

	client := c.getRawClient()
	res, err := client.Server.Reset().Find()
	if err != nil {
		return nil, err
	}
	return res.Servers, nil
}

func (c *client) ShutdownServerByID(id int64, shutdownWait time.Duration) error {
	client := c.getRawClient()

	// graceful shutdown(ACPI)
	if _, err := client.Server.Shutdown(id); err != nil {
		return err
	}
	if err := client.Server.SleepUntilDown(id, shutdownWait); err == nil {
		return nil
	}

	// force shutdown
	if _, err := client.Server.Stop(id); err == nil {
		return err
	}
	if err := client.Server.SleepUntilDown(id, shutdownWait); err == nil {
		return nil
	}

	return fmt.Errorf("shutdown server %q is failed", id)
}
