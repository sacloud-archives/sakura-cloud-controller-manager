package sakura

import (
	"github.com/sacloud/sakura-cloud-controller-manager/iaas"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/kubernetes/pkg/cloudprovider"
)

type zones struct {
	sacloudAPI iaas.Client
}

func newZones(client iaas.Client) *zones {
	return &zones{sacloudAPI: client}
}

// GetZone returns the Zone containing the current failure zone and locality region that the program is running in
// In most cases, this method is called from the kubelet querying a local metadata service to acquire its zone.
// For the case of external cloud providers, use GetZoneByProviderID or GetZoneByNodeName since GetZone
// can no longer be called from the kubelets.
func (z *zones) GetZone() (cloudprovider.Zone, error) {
	return cloudprovider.Zone{Region: z.sacloudAPI.CurrentZone()}, nil
}

// GetZoneByProviderID returns the Zone containing the current zone and locality region of the node specified by providerId
// This method is particularly used in the context of external cloud providers where node initialization must be down
// outside the kubelets.
func (z *zones) GetZoneByProviderID(providerID string) (cloudprovider.Zone, error) {
	id, err := serverIDFromProviderID(providerID)
	if err != nil {
		return cloudprovider.Zone{}, err
	}

	server, err := nodeByID(z.sacloudAPI, id)
	if err != nil {
		return cloudprovider.Zone{}, err
	}

	return cloudprovider.Zone{Region: server.Zone.Name}, nil
}

// GetZoneByNodeName returns the Zone containing the current zone and locality region of the node specified by node name
// This method is particularly used in the context of external cloud providers where node initialization must be down
// outside the kubelets.
func (z *zones) GetZoneByNodeName(nodeName types.NodeName) (cloudprovider.Zone, error) {
	server, err := nodeByName(z.sacloudAPI, string(nodeName))
	if err != nil {
		return cloudprovider.Zone{}, err
	}

	return cloudprovider.Zone{Region: server.Zone.Name}, nil
}
