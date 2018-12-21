package iaas

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadBalancerParam_assignAddressed(t *testing.T) {

	expects := []struct {
		cond     string
		start    string
		maskLen  int
		hasError bool
	}{
		{
			cond:     "192.2.0.1/24",
			start:    "192.2.0.0",
			maskLen:  24,
			hasError: false,
		},
		{
			cond:     "10.10.10.10",
			hasError: true,
		},
	}

	for _, expect := range expects {
		lbParam := &LoadBalancerParam{AssignIPAddressRange: expect.cond}
		start, maskLen, err := lbParam.assignAddresses()

		if !expect.hasError {
			assert.Equal(t, expect.start, start.String(), "assignAddresses: unexpected startAddress")
			assert.Equal(t, expect.maskLen, maskLen, "assignAddresses: unexpected maskLen")
		}
		assert.Equal(t, expect.hasError, err != nil, "assignAddresses: unexpected error")
	}
}
