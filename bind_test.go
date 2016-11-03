package registry

import (
	"github.com/stretchr/testify/assert"

	"testing"
)

type TestUser struct {
	ID        int    `default:"42"`
	Name      string `default:"test user name"`
	Anonymous struct {
		String string `default:"test user anonymous value"`
	}
}

type testConfig struct {
	TestUser
	Int         int      `default:"100"`
	String      string   `default:"StringVar" env:"STRING_VAR" flag:"string_var" registry:"string.var"`
	StringSlice []string `default:"a,b,c"`
	Float32     float32  `default:"36.6"`
	FlagString  string   `default:"-" flag:"string"`
	FlagInt     int      `default:"-" flag:"int"`
	Anonymous   struct {
		String string `default:"anonymous value"`
	}
	unexported struct {
		String string `default:"unexported"`
	}
}

func (*testConfig) Lock()   {}
func (*testConfig) Unlock() {}

func Test_Bind(t *testing.T) {
	args = []string{"--string=FlagString", "--int", "1000"}
	var (
		config   = testConfig{}
		registry = registry{}
	)
	if err := registry.Bind(&config); assert.NoError(t, err) {
		assert.Equal(t, int(100), config.Int)
		assert.Equal(t, "StringVar", config.String)
		assert.Equal(t, []string{"a", "b", "c"}, config.StringSlice)
		assert.Equal(t, float32(36.6), config.Float32)
		assert.Equal(t, "FlagString", config.FlagString)
		assert.Equal(t, int(1000), config.FlagInt)
		assert.Equal(t, "anonymous value", config.Anonymous.String)
		if assert.NotEqual(t, "unexported", config.unexported) {
			assert.Equal(t, int(42), config.TestUser.ID)
			assert.Equal(t, "test user name", config.TestUser.Name)
			assert.Equal(t, "test user anonymous value", config.TestUser.Anonymous.String)
		}
	}

	assert.Len(t, registry.configs, 1)
	args = []string{}
}
