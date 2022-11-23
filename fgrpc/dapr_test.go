package fgrpc

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseDaprParameters4InvokeAppCallback(t *testing.T) {
	d := &DaprGRPCRunnerResults{}
	err := d.parseDaprParameters("capability=invoke,target=appcallback,method=load,ex1=1,ex2=2")
	assert.NoError(t, err, "parse failed")
	assert.NotNil(t, d.params)
	assert.Equal(t, d.params.capability, "invoke")
	assert.Equal(t, d.params.target, "appcallback")
	assert.Equal(t, d.params.method, "load")
	assert.Equal(t, d.params.appId, "")
	assert.Equal(t, d.params.extensions["ex1"], "1")
	assert.Equal(t, d.params.extensions["ex2"], "2")
}

func TestParseDaprParameters4InvokeDapr(t *testing.T) {
	d := &DaprGRPCRunnerResults{}
	err := d.parseDaprParameters("capability=invoke,target=dapr,method=load,appid=testapp,ex1=1,ex2=2")
	assert.NoError(t, err, "parse failed")
	assert.NotNil(t, d.params)
	assert.Equal(t, d.params.capability, "invoke")
	assert.Equal(t, d.params.target, "dapr")
	assert.Equal(t, d.params.method, "load")
	assert.Equal(t, d.params.appId, "testapp")
	assert.Equal(t, d.params.extensions["ex1"], "1")
	assert.Equal(t, d.params.extensions["ex2"], "2")
}

func TestPrepareRequest4PubSub(t *testing.T) {
	t.Run("params sanity test", func(t *testing.T) {
		testcases := []struct {
			name    string
			params  string
			isError bool
			errSub  string
		}{
			{
				name:    "method is missing",
				params:  "capability=pubsub,target=dapr,store=memstore,topic=mytopic",
				isError: true,
				errSub:  "method is required",
			},
			{
				name:    "store is missing",
				params:  "capability=pubsub,target=dapr,method=publish,topic=mytopic",
				isError: true,
				errSub:  "store(pubsub name) is required",
			},
			{
				name:    "topic is missing",
				params:  "capability=pubsub,target=dapr,method=publish,store=memstore",
				isError: true,
				errSub:  "topic is required",
			},
			{
				name:    "method is invalid",
				params:  "capability=pubsub,target=dapr,method=invalid,store=memstore,topic=mytopic",
				isError: true,
				errSub:  "unsupported method",
			},
			{
				name:    "numevents is invalid",
				params:  "capability=pubsub,target=dapr,method=publish,store=memstore,topic=mytopic,numevents=invalid",
				isError: true,
				errSub:  "numevents must be integer",
			},
			{
				name:    "when method is bulkpublish, numevents is invalid",
				params:  "capability=pubsub,target=dapr,method=bulkpublish,store=memstore,topic=mytopic,numevents=invalid",
				isError: true,
				errSub:  "numevents must be integer",
			},
			{
				name:    "valid with method publish",
				params:  "capability=pubsub,target=dapr,method=publish,store=memstore,topic=mytopic",
				isError: false,
			},
			{
				name:    "valid with method bulkpublish",
				params:  "capability=pubsub,target=dapr,method=bulkpublish,store=memstore,topic=mytopic,numevents=100",
				isError: false,
			},
		}

		for _, tc := range testcases {
			t.Run(tc.name, func(t *testing.T) {
				o := &GRPCRunnerOptions{}
				o.UseDapr = tc.params

				d := &DaprGRPCRunnerResults{}
				err := d.parseDaprParameters(o.UseDapr)
				assert.NoError(t, err, "parse failed")

				err = d.prepareRequest4PubSub(o)
				if tc.isError {
					assert.Error(t, err, "should fail")
				} else {
					assert.NoError(t, err, "should succeed")
				}
			})
		}
	})

	t.Run("publish request", func(t *testing.T) {
		testcases := []struct {
			name      string
			numEvents int
		}{
			{
				name:      "numevents is missing",
				numEvents: -1,
			},
			{
				name:      "numevents is 1",
				numEvents: 1,
			},
			{
				name:      "numevents is 100",
				numEvents: 100,
			},
		}

		for _, tc := range testcases {
			t.Run(tc.name, func(t *testing.T) {
				o := &GRPCRunnerOptions{}
				o.UseDapr = "capability=pubsub,target=dapr,method=publish,store=memstore,topic=mytopic,contenttype=text/plain"
				if tc.numEvents > 0 {
					o.UseDapr += fmt.Sprintf(",numevents=%d", tc.numEvents)
				}
				o.Payload = "hello world"

				d := &DaprGRPCRunnerResults{}
				err := d.parseDaprParameters(o.UseDapr)
				assert.NoError(t, err, "parse failed")

				err = d.prepareRequest4PubSub(o)
				assert.NoError(t, err, "prepare failed")

				if tc.numEvents > 0 {
					assert.Equal(t, tc.numEvents, len(d.publishEventRequests))
				} else {
					assert.Equal(t, 1, len(d.publishEventRequests))
				}

				for _, req := range d.publishEventRequests {
					assert.Equal(t, "memstore", req.PubsubName)
					assert.Equal(t, "mytopic", req.Topic)
					assert.Equal(t, "text/plain", req.DataContentType)
					assert.Equal(t, "hello world", string(req.Data))
				}

				assert.Nil(t, d.bulkPublishRequest)
			})
		}
	})

	t.Run("bulk publish request", func(t *testing.T) {
		o := &GRPCRunnerOptions{}
		o.UseDapr = "capability=pubsub,target=dapr,method=bulkpublish,store=memstore,topic=mytopic,contenttype=text/plain,numevents=100"
		o.Payload = "hello world"

		d := &DaprGRPCRunnerResults{}
		err := d.parseDaprParameters(o.UseDapr)
		assert.NoError(t, err, "parse failed")

		err = d.prepareRequest4PubSub(o)
		assert.NoError(t, err, "prepare failed")

		assert.Equal(t, "memstore", d.bulkPublishRequest.PubsubName)
		assert.Equal(t, "mytopic", d.bulkPublishRequest.Topic)
		for _, entry := range d.bulkPublishRequest.Entries {
			assert.Equal(t, "text/plain", entry.ContentType)
			assert.Equal(t, "hello world", string(entry.Event))
			assert.NotEmpty(t, entry.EntryId)
		}

		assert.Nil(t, d.publishEventRequests)
	})
}
