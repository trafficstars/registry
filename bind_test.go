package registry

import (
	"time"

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
	Int8        int8     `default:"100"`
	Int16       int16    `default:"100"`
	Int32       int32    `default:"100"`
	Int64       int64    `default:"100"`
	Uint        uint     `default:"100"`
	Uint8       uint8    `default:"100"`
	Uint16      uint16   `default:"100"`
	Uint32      uint32   `default:"100"`
	Uint64      uint64   `default:"100"`
	String      string   `default:"StringVar" env:"STRING_VAR" flag:"string_var" registry:"string.var"`
	StringSlice []string `default:"a,b,c"`
	Float32     float32  `default:"36.6"`
	FlagString  string   `default:"-" flag:"string"`
	FlagInt     int      `default:"-" flag:"int"`
	Bool        bool     `default:"true"`
	Anonymous   struct {
		String string `default:"anonymous value"`
	}
	unexported struct {
		String string `default:"unexported"`
	}
	Duration time.Duration `default:"4m2s"`
}

func (*testConfig) Lock()   {}
func (*testConfig) Unlock() {}

func Test_Bind(t *testing.T) {
	args = []string{"--string=FlagString", "--int", "1000"}
	var (
		config   = testConfig{}
		registry = registry{
			bindChan: make(chan struct{}),
		}
	)
	go func() {
		for {
			<-registry.bindChan
		}
	}()
	if err := registry.Bind(&config); assert.NoError(t, err) {
		assert.Equal(t, int(100), config.Int)
		assert.Equal(t, int8(100), config.Int8)
		assert.Equal(t, int16(100), config.Int16)
		assert.Equal(t, int32(100), config.Int32)
		assert.Equal(t, int64(100), config.Int64)
		assert.Equal(t, uint(100), config.Uint)
		assert.Equal(t, uint8(100), config.Uint8)
		assert.Equal(t, uint16(100), config.Uint16)
		assert.Equal(t, uint32(100), config.Uint32)
		assert.Equal(t, uint64(100), config.Uint64)
		assert.Equal(t, "StringVar", config.String)
		assert.Equal(t, []string{"a", "b", "c"}, config.StringSlice)
		assert.Equal(t, float32(36.6), config.Float32)
		assert.Equal(t, "FlagString", config.FlagString)
		assert.Equal(t, int(1000), config.FlagInt)
		assert.Equal(t, "anonymous value", config.Anonymous.String)
		if assert.True(t, config.Bool) && assert.NotEqual(t, "unexported", config.unexported) {
			assert.Equal(t, int(42), config.TestUser.ID)
			assert.Equal(t, "test user name", config.TestUser.Name)
			assert.Equal(t, "test user anonymous value", config.TestUser.Anonymous.String)
		}
		assert.Equal(t, 4*time.Minute+2*time.Second, config.Duration)
	}

	assert.Len(t, registry.configs, 1)
	args = []string{}
}
