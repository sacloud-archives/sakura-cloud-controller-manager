package sakura

import (
	"context"
	"testing"

	"github.com/sacloud/libsacloud/sacloud"
	"github.com/stretchr/testify/assert"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/cloud-provider"
)

var dummyLBClient = &testSacloudClient{}
var dummyLoadbalancers = &loadbalancers{sacloudAPI: dummyLBClient}

func TestLoadBalancers_GetLoadBalancer(t *testing.T) {
	ctx := context.Background()
	svcUID := "test"
	service := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			UID: types.UID(svcUID),
		},
	}
	lbName := cloudprovider.DefaultLoadBalancerName(service)
	lb := newLoadBalancer(&newLoadBalancerParam{
		name:         lbName,
		availability: sacloud.EAAvailable,
		vips:         []string{"192.2.0.1"},
	})

	client := &testSacloudClient{
		loadBalancers: []sacloud.LoadBalancer{*lb},
	}
	lbs := &loadbalancers{sacloudAPI: client}

	status, exists, err := lbs.GetLoadBalancer(ctx, "test", service)
	assert.NotNil(t, status)
	assert.Len(t, status.Ingress, 1)
	assert.Equal(t, "192.2.0.1", status.Ingress[0].IP)
	assert.True(t, exists)
	assert.NoError(t, err)
}

type newLoadBalancerParam struct {
	name         string
	availability sacloud.EAvailability
	vips         []string
}

func newLoadBalancer(p *newLoadBalancerParam) *sacloud.LoadBalancer {
	values := &sacloud.CreateLoadBalancerValue{
		SwitchID:     "999",
		Name:         p.name,
		VRID:         1,
		Plan:         sacloud.LoadBalancerPlanStandard,
		IPAddress1:   "192.2.0.11",
		MaskLen:      24,
		DefaultRoute: "192.2.0.1",
	}
	var settings []*sacloud.LoadBalancerSetting
	for _, vip := range p.vips {
		settings = append(settings, &sacloud.LoadBalancerSetting{
			VirtualIPAddress: vip,
		})
	}

	lb, _ := sacloud.CreateNewLoadBalancerSingle(values, settings)
	lb.Availability = p.availability
	return lb
}
