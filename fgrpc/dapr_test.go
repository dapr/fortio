package fgrpc

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParseDaprParameters4InvokeAppCallback(t *testing.T) {
	d := &DaprGRPCRunnerResults{}
	err := d.parseDaprParameters("capability=invoke,target=appcallback,method=load,ex1=1,ex2=2")
	assert.NoError(t, err, "parse failed")
	assert.NotNil(t, d.params)
	assert.Equal(t, d.params.capability, "invoke")
	assert.Equal(t, d.params.target, "appcallback")
	assert.Equal(t, d.params.method, "load")
	assert.Equal(t, d.params.extensions["ex1"], "1")
	assert.Equal(t, d.params.extensions["ex2"], "2")
}

func TestParseDaprParameters4InvokeDapr(t *testing.T) {
	d := &DaprGRPCRunnerResults{}
	err := d.parseDaprParameters("capability=invoke,target=dapr,method=load,ex1=1,ex2=2")
	assert.NoError(t, err, "parse failed")
	assert.NotNil(t, d.params)
	assert.Equal(t, d.params.capability, "invoke")
	assert.Equal(t, d.params.target, "dapr")
	assert.Equal(t, d.params.method, "load")
	assert.Equal(t, d.params.extensions["ex1"], "1")
	assert.Equal(t, d.params.extensions["ex2"], "2")
}
