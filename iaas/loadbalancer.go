package iaas

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/sacloud/libsacloud/api"
	"github.com/sacloud/libsacloud/sacloud"
)

type globalIPList struct {
	gateway        string
	addresses      []string
	networkAddress string
	nwMaskLen      int
}

func (c *client) LoadBalancers(tags ...string) ([]sacloud.LoadBalancer, error) {
	lbClient := c.getRawClient().LoadBalancer.Reset()
	if len(tags) > 0 {
		lbClient.WithTags(tags)
	}
	res, err := lbClient.Find()
	if err != nil {
		return nil, err
	}
	return res.LoadBalancers, err
}

func (c *client) WaitForLBActive(id int64) error {
	return c.rawClient.LoadBalancer.SleepUntilUp(id, c.rawClient.DefaultTimeoutDuration)
}

func (c *client) CreateLoadBalancer(lbParam *LoadBalancerParam, vipParam *VIPParam) ([]string, error) {

	lock := sync.Mutex{}
	lock.Lock()
	var once sync.Once
	defer once.Do(lock.Unlock)

	client := c.getRawClient()
	sw, err := c.findLBConnectedSwitch(lbParam)
	if err != nil {
		return nil, err
	}
	if sw == nil {
		return nil, fmt.Errorf("switch resource (with tag[%s]) is not found", lbParam.RouterTags)
	}

	var globalIPs = []*globalIPList{}

	ips, err := sw.GetIPAddressList()
	if err != nil {
		return nil, err
	}
	globalIPs = append(globalIPs, &globalIPList{
		gateway:        sw.Subnets[0].DefaultRoute,
		networkAddress: sw.Subnets[0].NetworkAddress,
		addresses:      ips,
		nwMaskLen:      sw.Subnets[0].NetworkMaskLen,
	})
	usedIPs, err := c.extractConsumedGlobalIPs(sw.ID)
	if err != nil {
		return nil, err
	}

	usableIPSubnets := c.usableGlobalIPs(globalIPs, usedIPs)
	if len(usableIPSubnets) == 0 {
		return nil, errors.New("usable global-ip-address not found")
	}
	lbAlocatable := false
	var lbIP1, lbIP2, vip string

	lbSubnet := usableIPSubnets[0]
	if lbParam.UseHA {
		if len(lbSubnet.addresses) >= 3 { // for LB HA
			lbAlocatable = true
			lbIP1 = lbSubnet.addresses[0]
			lbIP2 = lbSubnet.addresses[1]
			vip = lbSubnet.addresses[2]
		}
	} else {
		if len(lbSubnet.addresses) >= 2 { // for LB single
			lbAlocatable = true
			lbIP1 = lbSubnet.addresses[0]
			vip = lbSubnet.addresses[1]
		}
	}
	if !lbAlocatable {
		return nil, errors.New("usable global-ip-address not found")
	}

	p := &sacloud.CreateLoadBalancerValue{
		SwitchID:     sw.GetStrID(),
		VRID:         1,
		Plan:         sacloud.LoadBalancerPlanStandard,
		IPAddress1:   lbIP1,
		MaskLen:      lbSubnet.nwMaskLen,
		DefaultRoute: lbSubnet.gateway,
		Name:         lbParam.Name,
		Description:  lbParam.Description,
		Tags:         lbParam.Tags,
	}
	if lbParam.UseHighSpecPlan {
		p.Plan = sacloud.LoadBalancerPlanPremium
	}

	settings := []*sacloud.LoadBalancerSetting{}
	for _, port := range vipParam.Ports {
		v := &sacloud.LoadBalancerSetting{
			VirtualIPAddress: vip,
			Port:             fmt.Sprintf("%d", port),
			DelayLoop:        fmt.Sprintf("%d", vipParam.HealthCheck.DelayLoop),
		}
		hc := &sacloud.LoadBalancerHealthCheck{
			Protocol: vipParam.HealthCheck.Protocol,
		}
		if vipParam.HealthCheck.Protocol == "http" || vipParam.HealthCheck.Protocol == "https" {
			hc.Path = vipParam.HealthCheck.Path
			hc.Status = fmt.Sprintf("%d", vipParam.HealthCheck.StatusCode)
		}

		for _, nodeIP := range vipParam.NodeIPs {
			v.AddServer(&sacloud.LoadBalancerServer{
				IPAddress:   nodeIP,
				Port:        fmt.Sprintf("%d", port),
				HealthCheck: hc,
				Enabled:     "True",
			})
		}
		settings = append(settings, v)
	}

	var createParam *sacloud.LoadBalancer
	if lbParam.UseHA {
		createParam, err = sacloud.CreateNewLoadBalancerDouble(&sacloud.CreateDoubleLoadBalancerValue{
			CreateLoadBalancerValue: p,
			IPAddress2:              lbIP2,
		}, settings)
	} else {
		createParam, err = sacloud.CreateNewLoadBalancerSingle(p, settings)
	}
	if err != nil {
		return nil, err
	}
	lb, err := client.LoadBalancer.Create(createParam)
	if err != nil {
		return nil, err
	}

	once.Do(lock.Unlock)

	err = c.waitForLoadBalancerBoot(client, lb.ID)
	if err != nil {
		return nil, err
	}
	_, err = client.LoadBalancer.Config(lb.ID)
	if err != nil {
		return nil, err
	}
	return []string{vip}, nil
}

func (c *client) waitForLoadBalancerBoot(client *api.Client, lbID int64) error {
	if err := client.LoadBalancer.SleepWhileCopying(lbID, client.DefaultTimeoutDuration, 20); err != nil {
		return fmt.Errorf("Failed to wait SakuraCloud LoadBalancer copy: %s", err)
	}
	if err := client.LoadBalancer.SleepUntilUp(lbID, client.DefaultTimeoutDuration); err != nil {
		return fmt.Errorf("Failed to wait SakuraCloud LoadBalancer boot: %s", err)
	}
	return nil
}

func (c *client) UpdateLoadBalancer(lb *sacloud.LoadBalancer, vipParam *VIPParam) ([]string, error) {
	settings := []*sacloud.LoadBalancerSetting{}
	vip := lb.Settings.LoadBalancer[0].VirtualIPAddress
	for _, port := range vipParam.Ports {
		v := &sacloud.LoadBalancerSetting{
			VirtualIPAddress: vip,
			Port:             fmt.Sprintf("%d", port),
			DelayLoop:        fmt.Sprintf("%d", vipParam.HealthCheck.DelayLoop),
		}
		hc := &sacloud.LoadBalancerHealthCheck{
			Protocol: vipParam.HealthCheck.Protocol,
		}
		if vipParam.HealthCheck.Protocol == "http" || vipParam.HealthCheck.Protocol == "https" {
			hc.Path = vipParam.HealthCheck.Path
			hc.Status = fmt.Sprintf("%d", vipParam.HealthCheck.StatusCode)
		}

		for _, nodeIP := range vipParam.NodeIPs {
			v.AddServer(&sacloud.LoadBalancerServer{
				IPAddress:   nodeIP,
				Port:        fmt.Sprintf("%d", port),
				HealthCheck: hc,
				Enabled:     "True",
			})
		}
		settings = append(settings, v)
	}

	lb.Settings.LoadBalancer = settings
	client := c.getRawClient()
	_, err := client.LoadBalancer.Update(lb.ID, lb)
	if err != nil {
		return nil, err
	}
	_, err = client.LoadBalancer.Config(lb.ID)
	if err != nil {
		return nil, err
	}

	return []string{vip}, nil
}

func (c *client) DeleteLoadBalancer(id int64) error {

	client := c.getRawClient()

	_, err := client.LoadBalancer.Stop(id)
	if err != nil {
		return err
	}
	err = client.LoadBalancer.SleepUntilDown(id, client.DefaultTimeoutDuration)
	if err != nil {
		return err
	}

	_, err = client.LoadBalancer.Delete(id)
	return err
}

func (c *client) findLBConnectedSwitch(lbParam *LoadBalancerParam) (*sacloud.Switch, error) {
	client := c.getRawClient()

	res, err := client.GetInternetAPI().Reset().WithTags(lbParam.RouterTags).Find()
	if err != nil {
		return nil, err
	}
	if len(res.Internet) > 0 {
		router := res.Internet[0]
		var sw *sacloud.Switch
		sw, err = client.Switch.Read(router.Switch.ID)
		if err != nil {
			return nil, err
		}
		return sw, nil
	}

	res, err = client.GetSwitchAPI().Reset().WithTags(lbParam.RouterTags).Find()
	if err != nil {
		return nil, err
	}
	if len(res.Switches) > 0 {
		return &res.Switches[0], nil
	}

	return nil, nil
}

func (c *client) extractConsumedGlobalIPs(routerSwitchID int64) ([]string, error) {

	var wg = sync.WaitGroup{}
	collectors := []func(*api.Client, int64) ([]string, error){
		c.extractConsumedIPsFromServer,
		c.extractConsumedIPsFromRouter,
		c.extractConsumedIPsFromLoadBalancer,
		c.extractConsumedIPsFromVPCRouter,
		c.extractConsumedIPsFromDB,
	}
	wg.Add(len(collectors))
	var resultChan = make(chan []string)
	var errChan = make(chan error)
	var done = make(chan bool)
	results := []string{}

	for _, v := range collectors {
		collector := v
		go func() {
			client := c.getRawClient()
			ips, err := collector(client, routerSwitchID)
			if err != nil {
				errChan <- err
				return
			}
			resultChan <- ips
		}()
	}

	// wait for extract complete
	go func() {
		wg.Wait()
		done <- true
	}()

	lock := sync.Mutex{}
	for {
		select {
		case ips := <-resultChan:
			lock.Lock()
			results = append(results, ips...)
			lock.Unlock()
			wg.Done()
		case err := <-errChan:
			return nil, err
		case <-done:
			return results, nil
		case <-time.After(time.Minute):
			return nil, fmt.Errorf("Extracting consumed GlobalIPs timed out")
		}
	}
}

func (c *client) extractConsumedIPsFromServer(client *api.Client, routerSwitchID int64) (ips []string, err error) {
	var res *sacloud.SearchResponse
	res, err = client.Server.Find()
	if err != nil {
		return
	}
	for _, server := range res.Servers {
		if len(server.Interfaces) > 0 {
			nic := server.Interfaces[0]
			if nic.Switch.ID == routerSwitchID {
				ips = append(ips, nic.UserIPAddress)
			}
		}
	}
	return
}

func (c *client) extractConsumedIPsFromRouter(client *api.Client, routerSwitchID int64) (ips []string, err error) {
	var sw *sacloud.Switch
	sw, err = client.Switch.Read(routerSwitchID)
	if err != nil {
		return
	}
	for _, subnet := range sw.Subnets {
		if subnet.NextHop != "" {
			ips = append(ips, subnet.NextHop)
		}
	}
	return
}

func (c *client) extractConsumedIPsFromLoadBalancer(client *api.Client, routerSwitchID int64) (ips []string, err error) {
	var res *api.SearchLoadBalancerResponse
	res, err = client.LoadBalancer.Find()
	if err != nil {
		return
	}
	for _, lb := range res.LoadBalancers {
		if lb.Switch.ID == routerSwitchID {
			for _, server := range lb.Remark.Servers {
				if ip, ok := server.(map[string]interface{})["IPAddress"]; ok {
					strIP := ip.(string)
					if strIP != "" {
						ips = append(ips, strIP)
					}
				}
			}
			// VIPs
			if lb.Settings != nil && lb.Settings.LoadBalancer != nil {
				for _, s := range lb.Settings.LoadBalancer {
					ips = append(ips, s.VirtualIPAddress)
				}
			}
		}
	}
	return
}

func (c *client) extractConsumedIPsFromVPCRouter(client *api.Client, routerSwitchID int64) (ips []string, err error) {
	var res *api.SearchVPCRouterResponse
	res, err = client.VPCRouter.Find()
	if err != nil {
		return
	}

	for _, vpcRouter := range res.VPCRouters {
		if vpcRouter.Interfaces[0].Switch.ID == routerSwitchID {
			nic := vpcRouter.Settings.Router.Interfaces[0]
			ips = append(ips, nic.IPAddress[0], nic.IPAddress[1])
			ips = append(ips, nic.IPAliases...)
		}
	}

	return
}

func (c *client) extractConsumedIPsFromDB(client *api.Client, routerSwitchID int64) (ips []string, err error) {
	var res *api.SearchDatabaseResponse
	res, err = client.Database.Find()
	if err != nil {
		return
	}
	for _, db := range res.Databases {
		if db.Switch.ID == routerSwitchID {
			for _, server := range db.Remark.Servers {
				if ip, ok := server.(map[string]interface{})["IPAddress"]; ok {
					strIP := ip.(string)
					if strIP != "" {
						ips = append(ips, strIP)
					}
				}
			}
		}
	}
	return
}

func (c *client) usableGlobalIPs(subnets []*globalIPList, usedIPs []string) []*globalIPList {
	res := []*globalIPList{}
	for _, subnet := range subnets {
		ips := []string{}
		for _, ip := range subnet.addresses {
			exists := false
			for _, v := range usedIPs {
				if v == ip {
					exists = true
					break
				}
			}
			if !exists {
				ips = append(ips, ip)
			}
		}
		if len(ips) > 0 {
			res = append(res, &globalIPList{
				gateway:        subnet.gateway,
				addresses:      ips,
				networkAddress: subnet.networkAddress,
				nwMaskLen:      subnet.nwMaskLen,
			})
		}
	}
	return res
}
