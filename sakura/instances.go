package sakura

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/sacloud/libsacloud/api"
	"github.com/sacloud/libsacloud/sacloud"
	"github.com/sacloud/sakura-cloud-controller-manager/iaas"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/cloud-provider"
)

type instances struct {
	sacloudAPI   iaas.Client
	shutdownWait time.Duration
}

const defaultServerShutdownWait = 30 * time.Second

func newInstances(client iaas.Client) cloudprovider.Instances {
	return &instances{
		sacloudAPI:   client,
		shutdownWait: defaultServerShutdownWait,
	}
}

// NodeAddresses returns the addresses of the specified instance.
func (i *instances) NodeAddresses(ctx context.Context, name types.NodeName) ([]v1.NodeAddress, error) {
	server, err := nodeByName(i.sacloudAPI, string(name))
	if err != nil {
		return nil, err
	}
	return nodeAddresses(server)
}

// NodeAddressesByProviderID returns the addresses of the specified instance.
// The instance is specified using the providerID of the node. The
// ProviderID is a unique identifier of the node. This will not be called
// from the node whose nodeaddresses are being queried. i.e. local metadata
// services cannot be used in this method to obtain nodeaddresses
//
// The providerID spec should be retrievable from the Kubernetes
// node object. The expected format is: sakuracloud://serverID
func (i *instances) NodeAddressesByProviderID(ctx context.Context, providerID string) ([]v1.NodeAddress, error) {
	serverID, err := serverIDFromProviderID(providerID)
	if err != nil {
		return nil, err
	}

	server, err := nodeByID(i.sacloudAPI, serverID)
	if err != nil {
		return nil, err
	}
	return nodeAddresses(server)
}

// ExternalID returns the cloud provider ID of the node with the specified NodeName.
// Note that if the instance does not exist or is no longer running, we must return ("", cloudprovider.InstanceNotFound)
func (i *instances) ExternalID(ctx context.Context, nodeName types.NodeName) (string, error) {
	return i.InstanceID(ctx, nodeName)
}

// InstanceID returns the cloud provider ID of the node with the specified NodeName.
func (i *instances) InstanceID(ctx context.Context, nodeName types.NodeName) (string, error) {
	server, err := nodeByName(i.sacloudAPI, string(nodeName))
	if err != nil {
		return "", err
	}
	return server.GetStrID(), nil
}

// InstanceType returns the type of the specified instance.
func (i *instances) InstanceType(ctx context.Context, name types.NodeName) (string, error) {
	server, err := nodeByName(i.sacloudAPI, string(name))
	if err != nil {
		return "", err
	}
	return strings.Replace(server.ServerPlan.ServiceClass, "/", "-", -1), nil
}

// InstanceTypeByProviderID returns the type of the specified instance.
func (i *instances) InstanceTypeByProviderID(ctx context.Context, providerID string) (string, error) {
	serverID, err := serverIDFromProviderID(providerID)
	if err != nil {
		return "", err
	}

	server, err := nodeByID(i.sacloudAPI, serverID)
	if err != nil {
		return "", err
	}
	return strings.Replace(server.ServerPlan.ServiceClass, "/", "-", -1), nil
}

// AddSSHKeyToAllInstances adds an SSH public key as a legal identity for all instances
// expected format for the key is standard ssh-keygen format: <protocol> <blob>
func (i *instances) AddSSHKeyToAllInstances(ctx context.Context, user string, keyData []byte) error {
	return cloudprovider.NotImplemented
}

// CurrentNodeName returns the name of the node we are currently running on
// On most clouds (e.g. GCE) this is the hostname, so we provide the hostname
func (i *instances) CurrentNodeName(ctx context.Context, hostname string) (types.NodeName, error) {
	return types.NodeName(hostname), nil
}

// InstanceExistsByProviderID returns true if the instance for the given provider id still is running.
// If false is returned with no error, the instance will be immediately deleted by the cloud controller manager.
func (i *instances) InstanceExistsByProviderID(ctx context.Context, providerID string) (bool, error) {
	serverID, err := serverIDFromProviderID(providerID)
	if err != nil {
		return false, err
	}

	_, err = nodeByID(i.sacloudAPI, serverID)
	if err == nil {
		return true, nil
	}
	if err == cloudprovider.InstanceNotFound {
		return false, nil
	}

	return false, err
}

// InstanceShutdownByProviderID returns true if the instance is shutdown in cloudprovider
func (i *instances) InstanceShutdownByProviderID(ctx context.Context, providerID string) (bool, error) {
	serverID, err := serverIDFromProviderID(providerID)
	if err != nil {
		return false, err
	}

	server, err := nodeByID(i.sacloudAPI, serverID)
	if err != nil {
		if err, ok := err.(api.Error); ok {
			if err.ResponseCode() == http.StatusNotFound {
				return false, cloudprovider.InstanceNotFound
			}
		}
		return false, err
	}

	if err := i.sacloudAPI.ShutdownServerByID(server.ID, i.shutdownWait); err != nil {
		return false, err
	}

	return true, nil
}

// nodeByName gets a SAKURA Cloud Server instance by name. The returned error will
// be cloudprovider.InstanceNotFound if the Server does not exist.
func nodeByName(client iaas.Client, name string) (*sacloud.Server, error) {
	servers, err := client.Servers()
	if err != nil {
		return nil, err
	}

	for _, server := range servers {
		if server.Name == name {
			return &server, nil
		}
	}

	return nil, cloudprovider.InstanceNotFound
}

// nodeByID gets a SAKURA Cloud Server instance by ID. The returned error will
// be cloudprovider.InstanceNotFound if the Server does not exist.
func nodeByID(client iaas.Client, id string) (*sacloud.Server, error) {
	servers, err := client.Servers()
	if err != nil {
		return nil, err
	}

	for _, server := range servers {
		if server.GetStrID() == id {
			return &server, nil
		}
	}

	return nil, cloudprovider.InstanceNotFound
}

func serverIDFromProviderID(providerID string) (string, error) {
	if providerID == "" {
		return "", errors.New("providerID cannot be empty string")
	}

	split := strings.Split(providerID, "/")
	if len(split) != 3 {
		return "", fmt.Errorf("unexpected providerID format: %s, format should be: sakuracloud://12345", providerID)
	}

	// since split[0] is actually "sakuracloud:"
	if strings.TrimSuffix(split[0], ":") != ProviderName {
		return "", fmt.Errorf("provider name from providerID should be sakuracloud: %s", providerID)
	}

	return split[2], nil
}

// nodeAddresses returns a []v1.NodeAddress from server.
func nodeAddresses(server *sacloud.Server) ([]v1.NodeAddress, error) {
	var addresses []v1.NodeAddress
	if len(server.Interfaces) > 0 && server.Interfaces[0].Switch != nil {
		switch server.Interfaces[0].Switch.Scope {
		case "shared":
			addresses = append(addresses, v1.NodeAddress{Type: v1.NodeExternalIP, Address: server.Interfaces[0].IPAddress})
			addresses = append(addresses, v1.NodeAddress{Type: v1.NodeInternalIP, Address: server.Interfaces[0].IPAddress})
		case "user":
			addresses = append(addresses, v1.NodeAddress{Type: v1.NodeExternalIP, Address: server.Interfaces[0].UserIPAddress})
			addresses = append(addresses, v1.NodeAddress{Type: v1.NodeInternalIP, Address: server.Interfaces[0].UserIPAddress})
		}
	}
	return addresses, nil
}
