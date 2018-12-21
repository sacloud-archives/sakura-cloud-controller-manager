package iaas

import (
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/sacloud/libsacloud/sacloud"
)

const (
	// LoadBalancerTypesInternet represents Router+Switch connected LoadBalancer
	LoadBalancerTypesInternet = "internet"
	// LoadBalancerTypesSwitch represents Switch connected LoadBalancer
	LoadBalancerTypesSwitch = "switch"
)

type globalIPList struct {
	gateway        string
	addresses      []string
	networkAddress string
	nwMaskLen      int
}

type loadBalancerIPs struct {
	switchID  int64
	vrid      int
	vip       string
	ip1       string
	ip2       string
	nwMaskLen int
	gateway   string
}

func (c *client) LoadBalancers(tags ...string) ([]sacloud.LoadBalancer, error) {
	return c.getAPIClient().FindLoadBalancersByTags()
}

func (c *client) WaitForLBActive(id int64, waitTimeout time.Duration) error {
	return c.getAPIClient().WaitForLBActive(id, waitTimeout)
}

func (c *client) CreateLoadBalancer(lbParam *LoadBalancerParam, vipParam *VIPParam, waitTimeout time.Duration) ([]string, error) {

	lock := sync.Mutex{}
	lock.Lock()
	var once sync.Once
	defer once.Do(lock.Unlock)

	client := c.getAPIClient()
	lbIPs, err := c.extractLoadBalancerIPSettings(lbParam)
	if err != nil {
		return nil, err
	}

	p := &sacloud.CreateLoadBalancerValue{
		SwitchID:     fmt.Sprintf("%d", lbIPs.switchID),
		VRID:         lbIPs.vrid,
		Plan:         sacloud.LoadBalancerPlanStandard,
		IPAddress1:   lbIPs.ip1,
		MaskLen:      lbIPs.nwMaskLen,
		DefaultRoute: lbIPs.gateway,
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
			VirtualIPAddress: lbIPs.vip,
			Port:             fmt.Sprintf("%d", port.Port),
			DelayLoop:        fmt.Sprintf("%d", port.HealthCheck.DelayLoop),
		}
		hc := &sacloud.LoadBalancerHealthCheck{
			Protocol: port.HealthCheck.Protocol,
		}

		// we don't use this(for future)
		if port.HealthCheck.Protocol == "http" || port.HealthCheck.Protocol == "https" {
			hc.Path = port.HealthCheck.Path
			hc.Status = fmt.Sprintf("%d", port.HealthCheck.StatusCode)
		}

		for _, nodeIP := range vipParam.NodeIPs {
			v.AddServer(&sacloud.LoadBalancerServer{
				IPAddress:   nodeIP,
				Port:        fmt.Sprintf("%d", port.HealthCheck.Port),
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
			IPAddress2:              lbIPs.ip2,
		}, settings)
	} else {
		createParam, err = sacloud.CreateNewLoadBalancerSingle(p, settings)
	}
	if err != nil {
		return nil, err
	}
	lb, err := client.CreateLoadBalancer(createParam)
	if err != nil {
		return nil, err
	}

	once.Do(lock.Unlock)

	err = client.WaitForLBActive(lb.ID, waitTimeout)
	if err != nil {
		return nil, err
	}
	if err = client.ApplyLoadBalancerConfig(lb.ID); err != nil {
		return nil, err
	}
	return []string{lbIPs.vip}, nil
}

func (c *client) UpdateLoadBalancer(lb *sacloud.LoadBalancer, lbParam *LoadBalancerParam, vipParam *VIPParam) ([]string, error) {
	var settings []*sacloud.LoadBalancerSetting

	vip := lb.Settings.LoadBalancer[0].VirtualIPAddress
	// Check VIP duplication if specified
	if vipParam.VIP != "" && vipParam.VIP != lb.Settings.LoadBalancer[0].VirtualIPAddress {
		var sw *sacloud.Switch
		var err error
		var assignableIPInfo *assignableIPInfo

		switch vipParam.Type {
		case LoadBalancerTypesInternet:
			sw, err = c.findLBConnectedRouterSwitch(lbParam)
			if err != nil {
				return nil, err
			}
			if sw == nil {
				return nil, fmt.Errorf("switch resource (with tag[%s]) is not found", lbParam.RouterTags)
			}
			assignableIPInfo, err = c.extractAssignableExternalIPs(lbParam, sw)
			if err != nil {
				return nil, err
			}

		case LoadBalancerTypesSwitch:
			sw, err = c.findLBConnectedSwitch(lbParam)
			if err != nil {
				return nil, err
			}
			if sw == nil {
				return nil, fmt.Errorf("switch resource (with tag[%s]) is not found", lbParam.RouterTags)
			}
			assignableIPInfo, err = c.extractAssignableInternalIPs(lbParam, sw)
			if err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("invalid LoadBalancerType %q is specified", vipParam.Type)
		}

		found := false
		for _, addr := range assignableIPInfo.assignableIPs {
			if addr == vipParam.VIP {
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("can't use specified VIP %q", vipParam.VIP)
		}

		vip = vipParam.VIP
	}

	for _, port := range vipParam.Ports {
		v := &sacloud.LoadBalancerSetting{
			VirtualIPAddress: vip,
			Port:             fmt.Sprintf("%d", port.Port),
			DelayLoop:        fmt.Sprintf("%d", port.HealthCheck.DelayLoop),
		}
		hc := &sacloud.LoadBalancerHealthCheck{
			Protocol: port.HealthCheck.Protocol,
		}
		if port.HealthCheck.Protocol == "http" || port.HealthCheck.Protocol == "https" {
			hc.Path = port.HealthCheck.Path
			hc.Status = fmt.Sprintf("%d", port.HealthCheck.StatusCode)
		}

		for _, nodeIP := range vipParam.NodeIPs {
			v.AddServer(&sacloud.LoadBalancerServer{
				IPAddress:   nodeIP,
				Port:        fmt.Sprintf("%d", port.HealthCheck.Port),
				HealthCheck: hc,
				Enabled:     "True",
			})
		}
		settings = append(settings, v)
	}

	lb.Settings.LoadBalancer = settings
	if _, err := c.apiClient.UpdateLoadBalancer(lb.ID, lb); err != nil {
		return nil, err
	}
	if err := c.apiClient.ApplyLoadBalancerConfig(lb.ID); err != nil {
		return nil, err
	}

	return []string{vip}, nil
}

func (c *client) DeleteLoadBalancer(id int64, waitTimeout time.Duration) error {
	return c.getAPIClient().DeleteLoadBalancer(id, waitTimeout)
}

func (c *client) extractLoadBalancerIPSettings(lbParam *LoadBalancerParam) (*loadBalancerIPs, error) {
	var sw *sacloud.Switch
	var vrid int
	var assignableIPInfo *assignableIPInfo
	var err error

	switch lbParam.Type {
	case LoadBalancerTypesInternet:
		sw, err = c.findLBConnectedRouterSwitch(lbParam)
		if err != nil {
			return nil, err
		}
		if sw == nil {
			return nil, fmt.Errorf("switch resource (with tag[%s]) is not found", lbParam.RouterTags)
		}

		// VRID scope = clusterID
		vrid, err = c.selectUniqVRID(lbParam, sw.ID)
		if err != nil {
			return nil, err
		}

		// get assignable IPs
		assignableIPInfo, err = c.extractAssignableExternalIPs(lbParam, sw)
		if err != nil {
			return nil, err
		}

	case LoadBalancerTypesSwitch:
		sw, err = c.findLBConnectedSwitch(lbParam)
		if err != nil {
			return nil, err
		}
		if sw == nil {
			return nil, fmt.Errorf("switch resource (with tag[%s]) is not found", lbParam.RouterTags)
		}

		// VRID scope = clusterID
		vrid, err = c.selectUniqVRID(lbParam, sw.ID)
		if err != nil {
			return nil, err
		}

		// get assignable IPs
		assignableIPInfo, err = c.extractAssignableInternalIPs(lbParam, sw)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("Invalid LoadBalancer type %q is specidied", lbParam.Type)
	}

	vip, lbIP1, lbIP2, err := c.choiceLoadBalancerIPs(lbParam, assignableIPInfo.assignableIPs, assignableIPInfo.usedIPs)
	if err != nil {
		return nil, err
	}

	// check VIP
	_, lbIPNet, err := net.ParseCIDR(fmt.Sprintf("%s/%d", lbIP1, assignableIPInfo.maskLen))
	if err != nil {
		return nil, err
	}
	parsedVIP, _, err := net.ParseCIDR(fmt.Sprintf("%s/%d", vip, assignableIPInfo.maskLen))
	if err != nil {
		return nil, err
	}
	if !lbIPNet.Contains(parsedVIP) {
		return nil, fmt.Errorf("VIP %q must be in same LB network %q", vip, lbIPNet.String())
	}

	return &loadBalancerIPs{
		switchID:  sw.ID,
		vrid:      vrid,
		vip:       vip,
		ip1:       lbIP1,
		ip2:       lbIP2,
		nwMaskLen: assignableIPInfo.maskLen,
		gateway:   assignableIPInfo.gateway,
	}, nil

}

func (c *client) choiceLoadBalancerIPs(lbParam *LoadBalancerParam, assignableIPs []string, usedIPs []string) (string, string, string, error) {
	lbAllocatable := false
	var lbIP1, lbIP2, vip string
	if lbParam.VIP != "" {
		// validate vip
		for _, used := range usedIPs {
			if used == lbParam.VIP {
				return "", "", "", fmt.Errorf("VIP %q is already used", lbParam.VIP)
			}
		}
		vip = lbParam.VIP
	}

	if lbParam.UseHA {
		reqIPNum := 3
		if lbParam.VIP != "" {
			reqIPNum = 2
		}
		if len(assignableIPs) >= reqIPNum { // for LB HA
			lbAllocatable = true
			lbIP1 = assignableIPs[0]
			lbIP2 = assignableIPs[1]
			if vip == "" {
				vip = assignableIPs[2]
			}
		}
	} else {
		reqIPNum := 2
		if lbParam.VIP != "" {
			reqIPNum = 1
		}
		if len(assignableIPs) >= reqIPNum { // for LB single
			lbAllocatable = true
			lbIP1 = assignableIPs[0]
			if vip == "" {
				vip = assignableIPs[1]
			}
		}
	}
	if !lbAllocatable {
		return "", "", "", errors.New("usable ip-address not found")
	}

	return vip, lbIP1, lbIP2, nil
}

type assignableIPInfo struct {
	assignableIPs []string
	usedIPs       []string
	maskLen       int
	gateway       string
}

func (c *client) extractAssignableInternalIPs(lbParam *LoadBalancerParam, sw *sacloud.Switch) (*assignableIPInfo, error) {
	// IPAddress scope = clusterID
	usedIPs, err := c.extractConsumedIPsFromLoadBalancerWithSwitch(lbParam, sw.ID)
	if err != nil {
		return nil, err
	}

	// parse LoadBalancer network options
	assignIP, poolSize, err := lbParam.assignAddresses()
	if err != nil {
		return nil, err
	}

	// search assignable IPAddress
	var assignableIPs []string
	baseIPPart := assignIP[3]
	for i := 1; len(assignableIPs) < 3 && i < poolSize; i++ {
		// Note: current implementation is not support lager than /24 subnet for assign IP
		assignIP[3] = baseIPPart + byte(i)
		found := false
		for _, used := range usedIPs {
			if used == assignIP.String() {
				found = true
				break
			}
		}
		if !found {
			assignableIPs = append(assignableIPs, assignIP.String())
		}
	}

	maskLen, err := lbParam.nwMaskLen()
	if err != nil {
		return nil, err
	}

	return &assignableIPInfo{
		assignableIPs: assignableIPs,
		usedIPs:       usedIPs,
		maskLen:       maskLen,
		gateway:       lbParam.DefaultGateway,
	}, nil
}

func (c *client) extractAssignableExternalIPs(_ *LoadBalancerParam, routerConnectedSwitch *sacloud.Switch) (*assignableIPInfo, error) {
	var globalIPs []*globalIPList
	ips, err := routerConnectedSwitch.GetIPAddressList()
	if err != nil {
		return nil, err
	}
	globalIPs = append(globalIPs, &globalIPList{
		gateway:        routerConnectedSwitch.Subnets[0].DefaultRoute,
		networkAddress: routerConnectedSwitch.Subnets[0].NetworkAddress,
		addresses:      ips,
		nwMaskLen:      routerConnectedSwitch.Subnets[0].NetworkMaskLen,
	})
	usedIPs, err := c.extractConsumedGlobalIPs(routerConnectedSwitch.ID)
	if err != nil {
		return nil, err
	}

	usableIPSubnets := c.usableGlobalIPs(globalIPs, usedIPs)
	if len(usableIPSubnets) == 0 {
		return nil, errors.New("usable global-ip-address not found")
	}
	lbSubnet := usableIPSubnets[0]
	var assignableIPs []string
	for _, addr := range lbSubnet.addresses {
		assignableIPs = append(assignableIPs, addr)
	}
	return &assignableIPInfo{
		assignableIPs: assignableIPs,
		usedIPs:       usedIPs,
		maskLen:       lbSubnet.nwMaskLen,
		gateway:       lbSubnet.gateway,
	}, nil
}

func (c *client) findLBConnectedSwitch(lbParam *LoadBalancerParam) (*sacloud.Switch, error) {

	switches, err := c.apiClient.FindSwitchesByTags(lbParam.RouterTags...)
	if err != nil {
		return nil, err
	}
	if len(switches) > 0 {
		return &switches[0], nil
	}

	return nil, nil
}

func (c *client) findLBConnectedRouterSwitch(lbParam *LoadBalancerParam) (*sacloud.Switch, error) {

	routers, err := c.apiClient.FindRoutersByTags(lbParam.RouterTags...)
	if err != nil {
		return nil, err
	}
	if len(routers) > 0 {
		router := routers[0]
		var sw *sacloud.Switch
		sw, err = c.apiClient.ReadSwitch(router.Switch.ID)
		if err != nil {
			return nil, err
		}
		return sw, nil
	}

	switches, err := c.apiClient.FindSwitchesByTags(lbParam.RouterTags...)
	if err != nil {
		return nil, err
	}
	if len(switches) > 0 {
		return &switches[0], nil
	}

	return nil, nil
}

func (c *client) extractConsumedGlobalIPs(routerSwitchID int64) ([]string, error) {

	var wg = sync.WaitGroup{}
	collectors := []func(apiClient, int64) ([]string, error){
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
			client := c.getAPIClient()
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

func (c *client) extractConsumedIPsFromServer(client apiClient, routerSwitchID int64) (ips []string, err error) {
	servers, err := client.FindServers()
	if err != nil {
		return
	}
	for _, server := range servers {
		if len(server.Interfaces) > 0 {
			nic := server.Interfaces[0]
			if nic.Switch.ID == routerSwitchID {
				ips = append(ips, nic.UserIPAddress)
			}
		}
	}
	return
}

func (c *client) extractConsumedIPsFromRouter(client apiClient, routerSwitchID int64) (ips []string, err error) {
	var sw *sacloud.Switch
	sw, err = client.ReadSwitch(routerSwitchID)
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

func (c *client) extractConsumedIPsFromLoadBalancer(client apiClient, routerSwitchID int64) (ips []string, err error) {
	lbs, err := client.FindLoadBalancers()
	if err != nil {
		return
	}
	for _, lb := range lbs {
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

func (c *client) selectUniqVRID(lbParam *LoadBalancerParam, switchID int64) (int, error) {
	lbs, err := c.apiClient.FindLoadBalancersByTags(lbParam.ClusterSelector...)
	if err != nil {
		return -1, err
	}
	var vrids []int
	for _, lb := range lbs {
		if lb.Switch.ID == switchID {
			vrids = append(vrids, lb.Remark.VRRP.VRID)
		}
	}

	for vrid := 1; ; vrid++ {
		found := false
		for _, used := range vrids {
			if used == vrid {
				found = true
				break
			}
		}
		if !found {
			return vrid, nil
		}
	}
}

func (c *client) extractConsumedIPsFromLoadBalancerWithSwitch(lbParam *LoadBalancerParam, switchID int64) (ips []string, err error) {
	lbs, err := c.apiClient.FindLoadBalancersByTags(lbParam.ClusterSelector...)
	if err != nil {
		return
	}
	for _, lb := range lbs {
		if lb.Switch.ID == switchID {
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

func (c *client) extractConsumedIPsFromVPCRouter(client apiClient, routerSwitchID int64) (ips []string, err error) {
	vpcRouters, err := client.FindVPCRouters()
	if err != nil {
		return
	}

	for _, vpcRouter := range vpcRouters {
		if vpcRouter.Interfaces[0].Switch.ID == routerSwitchID {
			nic := vpcRouter.Settings.Router.Interfaces[0]
			ips = append(ips, nic.IPAddress[0], nic.IPAddress[1])
			ips = append(ips, nic.IPAliases...)
		}
	}

	return
}

func (c *client) extractConsumedIPsFromDB(client apiClient, routerSwitchID int64) (ips []string, err error) {
	dbs, err := client.FindDatabases()
	if err != nil {
		return
	}
	for _, db := range dbs {
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
	var res []*globalIPList
	for _, subnet := range subnets {
		var ips []string
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
