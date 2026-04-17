package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPKCEHelper_GeneratePKCE(t *testing.T) {
	helper := &PKCEHelper{}

	verifier, challenge, err := helper.GeneratePKCE()
	assert.NoError(t, err)
	assert.NotEmpty(t, verifier)
	assert.NotEmpty(t, challenge)
	assert.NotEqual(t, verifier, challenge)
}

func TestPKCEHelper_GenerateState(t *testing.T) {
	helper := &PKCEHelper{}

	state, err := helper.GenerateState()
	assert.NoError(t, err)
	assert.NotEmpty(t, state)
}

func TestPKCEHelper_GenerateUUID(t *testing.T) {
	helper := &PKCEHelper{}

	uuid, err := helper.GenerateUUID()
	assert.NoError(t, err)
	assert.NotEmpty(t, uuid)
	// 验证 UUID 格式
	assert.Len(t, uuid, 36) // UUID 格式: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
}
