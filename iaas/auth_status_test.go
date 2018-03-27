package iaas

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestE2EAuthStatus(t *testing.T) {
	res, err := testClient.AuthStatus()
	assert.NoError(t, err)
	assert.NotNil(t, res)
}
