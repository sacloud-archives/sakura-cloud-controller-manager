package iaas

import "github.com/sacloud/libsacloud/sacloud"

func (c *client) AuthStatus() (*sacloud.AuthStatus, error) {
	return c.rawClient.AuthStatus.Read()
}
