package logger

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func resetRegistry() {
	registryMu.Lock()
	registryByName = make(map[string]Logger)
	registryMu.Unlock()
}

func TestRegisterAndGet(t *testing.T) {
	resetRegistry()

	err := Register("", Nop())
	assert.Equal(t, errEmptyLoggerName, err)

	err = Register("test", nil)
	assert.Equal(t, errNilLogger, err)

	err = Register("test", Nop())
	require.NoError(t, err)

	err = Register("test", Nop())
	assert.Equal(t, errLoggerRegistered, err)

	got := Get("test")
	assert.Equal(t, Nop(), got)

	missing := Get("missing")
	_, ok := missing.(nopLogger)
	assert.True(t, ok)
}

func TestNames(t *testing.T) {
	resetRegistry()

	require.NoError(t, Register("a", Nop()))
	require.NoError(t, Register("b", Nop()))

	names := Names()
	assert.Len(t, names, 2)
	assert.Contains(t, names, "a")
	assert.Contains(t, names, "b")
}
