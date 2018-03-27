package iaas

import "github.com/sacloud/libsacloud/sacloud"

func (c *client) Servers() ([]sacloud.Server, error) {

	client := c.getRawClient()
	res, err := client.Server.Reset().Find()
	if err != nil {
		return nil, err
	}
	return res.Servers, nil
}
