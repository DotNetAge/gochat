package pipeline

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestState_GetString(t *testing.T) {
	state := NewState()
	state.Set("key", "value")

	val, ok := state.GetString("key")
	assert.True(t, ok)
	assert.Equal(t, "value", val)

	_, ok2 := state.GetString("nonexistent")
	assert.False(t, ok2)

	state.Set("wrongtype", 123)
	_, ok3 := state.GetString("wrongtype")
	assert.False(t, ok3)
}

func TestState_GetInt(t *testing.T) {
	state := NewState()
	state.Set("intKey", 42)

	val, ok := state.GetInt("intKey")
	assert.True(t, ok)
	assert.Equal(t, 42, val)

	val2, ok2 := state.GetInt("nonexistent")
	assert.False(t, ok2)
	assert.Equal(t, 0, val2)
}

func TestState_GetFloat(t *testing.T) {
	state := NewState()
	state.Set("floatKey", 3.14)

	val, ok := state.GetFloat("floatKey")
	assert.True(t, ok)
	assert.Equal(t, 3.14, val)

	val2, ok2 := state.GetFloat("nonexistent")
	assert.False(t, ok2)
	assert.Equal(t, 0.0, val2)
}

func TestState_GetBool(t *testing.T) {
	state := NewState()
	state.Set("boolKey", true)

	val, ok := state.GetBool("boolKey")
	assert.True(t, ok)
	assert.True(t, val)

	state.Set("falseKey", false)
	val2, ok2 := state.GetBool("falseKey")
	assert.True(t, ok2)
	assert.False(t, val2)

	val3, ok3 := state.GetBool("nonexistent")
	assert.False(t, ok3)
	assert.False(t, val3)
}

func TestState_GetStringSlice(t *testing.T) {
	state := NewState()
	slice := []string{"a", "b", "c"}
	state.Set("sliceKey", slice)

	val, ok := state.GetStringSlice("sliceKey")
	assert.True(t, ok)
	assert.Equal(t, slice, val)

	val2, ok2 := state.GetStringSlice("nonexistent")
	assert.False(t, ok2)
	assert.Nil(t, val2)
}

func TestState_GetMap(t *testing.T) {
	state := NewState()
	m := map[string]interface{}{"key": "value"}
	state.Set("mapKey", m)

	val, ok := state.GetMap("mapKey")
	assert.True(t, ok)
	assert.Equal(t, m, val)

	val2, ok2 := state.GetMap("nonexistent")
	assert.False(t, ok2)
	assert.Nil(t, val2)
}

func TestState_TypeConversion(t *testing.T) {
	state := NewState()
	state.Set("intAsFloat", 42)

	_, ok := state.GetFloat("intAsFloat")
	assert.False(t, ok)

	state.Set("floatAsInt", 3.14)
	_, ok = state.GetInt("floatAsInt")
	assert.False(t, ok)
}
