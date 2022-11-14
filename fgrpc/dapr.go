package fgrpc

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	v1 "github.com/dapr/dapr/pkg/proto/common/v1"
	dapr "github.com/dapr/dapr/pkg/proto/runtime/v1"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/anypb"
)

const CAPABILITY_INVOKE = "invoke"
const CAPABILITY_STATE = "state"
const CAPABILITY_PUBSUB = "pubsub"
const TARGET_NOOP = "noop"
const TARGET_DAPR = "dapr"
const TARGET_APPCALLBACK = "appcallback"

type DaprGRPCRunnerResults struct {
	// common
	params            *DaprRequestParameters
	daprClient        dapr.DaprClient
	appCallbackClient dapr.AppCallbackClient

	// service invoke
	invokeRequest            *dapr.InvokeServiceRequest
	invokeAppCallbackRequest *v1.InvokeRequest

	// state
	getStateRequest *dapr.GetStateRequest

	// pub-sub
	publishEventRequest  *dapr.PublishEventRequest
	publishEventRequests []*dapr.PublishEventRequest
	bulkPublishRequest   *dapr.BulkPublishRequest
}

type DaprRequestParameters struct {
	capability string
	target     string
	method     string
	appId      string
	store      string

	extensions map[string]string
}

func (d *DaprGRPCRunnerResults) PrepareRequestAndConnection(o *GRPCRunnerOptions, conn *grpc.ClientConn) error {
	err := d.parseDaprParameters(o.UseDapr)
	if err != nil {
		return err
	}

	t := d.params.target
	c := d.params.capability
	err = fmt.Errorf("unsupported dapr load test: capability=%s, target=%s", c, t)

	if t == TARGET_NOOP {
		// do nothing for no-op
		return nil
	} else if t == TARGET_DAPR {
		d.daprClient = dapr.NewDaprClient(conn)
		if c == CAPABILITY_INVOKE {
			err = d.prepareRequest4Invoke(o)
		} else if c == CAPABILITY_STATE {
			err = d.prepareRequest4State(o)
		} else if c == CAPABILITY_PUBSUB {
			err = d.prepareRequest4PubSub(o)
		}
	} else if t == TARGET_APPCALLBACK {
		d.appCallbackClient = dapr.NewAppCallbackClient(conn)
		if c == CAPABILITY_INVOKE {
			err = d.prepareRequest4InvokeAppCallback(o)
		}
	}

	return err
}

func (d *DaprGRPCRunnerResults) prepareRequest4Invoke(o *GRPCRunnerOptions) error {
	method := d.params.method
	if method == "" {
		return fmt.Errorf("method is required for load test")
	}

	d.invokeRequest = &dapr.InvokeServiceRequest{
		Id: d.params.appId,
		Message: &v1.InvokeRequest{
			Method:      method,
			ContentType: "text/plain",
		},
	}

	if len(o.Payload) > 0 {
		d.invokeRequest.Message.Data = &anypb.Any{Value: []byte(o.Payload)}
	} else {
		d.invokeRequest.Message.Data = &anypb.Any{Value: []byte{}}
	}
	return nil
}

func (d *DaprGRPCRunnerResults) prepareRequest4State(o *GRPCRunnerOptions) error {
	method := d.params.method
	store := d.params.store
	key := d.params.extensions["key"]
	if method == "" {
		return fmt.Errorf("method is required for state load test")
	}
	if store == "" {
		return fmt.Errorf("store is required for state load test")
	}
	if key == "" {
		return fmt.Errorf("key is required for state load test")
	}

	switch method {
	case "get":
		d.getStateRequest = &dapr.GetStateRequest{
			StoreName: store,
			Key:       key,
		}
	default:
		return fmt.Errorf("unsupported method of state load test: method=%s", method)
	}

	return nil
}

func (d *DaprGRPCRunnerResults) prepareRequest4PubSub(o *GRPCRunnerOptions) error {
	method := d.params.method
	store := d.params.store
	topic := d.params.extensions["topic"]
	contentType := d.params.extensions["contenttype"]
	metadata := stringToMap(d.params.extensions["metadata"])
	numEvents := d.params.extensions["numevents"]
	numEventsInt := 0

	if method == "" {
		return fmt.Errorf("method is required for pubsub load test")
	}
	if store == "" {
		return fmt.Errorf("store(pubsub name) is required for pubsub load test")
	}
	if topic == "" {
		return fmt.Errorf("topic is required for pubsub load test")
	}
	if method != "publish" {
		if numEvents == "" {
			return fmt.Errorf("numevents is required for pubsub load test when method is %s", method)
		}

		var err error
		numEventsInt, err = strconv.Atoi(numEvents)
		if err != nil {
			return fmt.Errorf("numevents must be integer: count=%s", numEvents)
		}
	}

	switch method {
	case "publish":
		d.publishEventRequest = &dapr.PublishEventRequest{
			PubsubName:      store,
			Topic:           topic,
			DataContentType: contentType,
			Metadata:        metadata,
		}
		if len(o.Payload) > 0 {
			d.publishEventRequest.Data = []byte(o.Payload)
		} else {
			d.publishEventRequest.Data = []byte{}
		}
	case "publish-multi":
		d.publishEventRequests = make([]*dapr.PublishEventRequest, numEventsInt)
		for i := 0; i < numEventsInt; i++ {
			d.publishEventRequests[i] = &dapr.PublishEventRequest{
				PubsubName:      store,
				Topic:           topic,
				DataContentType: contentType,
				Metadata:        metadata,
			}
			if len(o.Payload) > 0 {
				d.publishEventRequests[i].Data = []byte(o.Payload)
			} else {
				d.publishEventRequests[i].Data = []byte{}
			}
		}
	case "bulkpublish":
		d.bulkPublishRequest = &dapr.BulkPublishRequest{
			PubsubName: store,
			Topic:      topic,
		}
		d.bulkPublishRequest.Entries = make([]*dapr.BulkPublishRequestEntry, numEventsInt)
		for i := 0; i < numEventsInt; i++ {
			d.bulkPublishRequest.Entries[i] = &dapr.BulkPublishRequestEntry{
				EntryId:     strconv.Itoa(i),
				ContentType: contentType,
				Metadata:    metadata,
			}
			if len(o.Payload) > 0 {
				d.bulkPublishRequest.Entries[i].Event = []byte(o.Payload)
			} else {
				d.bulkPublishRequest.Entries[i].Event = []byte{}
			}
		}
	default:
		return fmt.Errorf("unsupported method of pubsub load test: method=%s", method)
	}

	return nil
}

func (d *DaprGRPCRunnerResults) prepareRequest4InvokeAppCallback(o *GRPCRunnerOptions) error {
	method := d.params.method
	if method == "" {
		return fmt.Errorf("method is required for load test")
	}

	d.invokeAppCallbackRequest = &v1.InvokeRequest{
		Method:      method,
		ContentType: "text/plain",
	}
	if len(o.Payload) > 0 {
		d.invokeAppCallbackRequest.Data = &anypb.Any{Value: []byte(o.Payload)}
	} else {
		d.invokeAppCallbackRequest.Data = &anypb.Any{Value: []byte{}}
	}
	return nil
}

func (d *DaprGRPCRunnerResults) RunTest() error {
	t := d.params.target
	c := d.params.capability
	m := d.params.method
	if t == TARGET_NOOP {
		// do nothing for no-op
		return nil
	}

	err := fmt.Errorf("unsupported dapr load test: capability=%s, target=%s, method=%s", c, t, m)

	if c == CAPABILITY_INVOKE {
		if t == TARGET_DAPR {
			_, err = d.daprClient.InvokeService(context.Background(), d.invokeRequest)
		} else if t == TARGET_APPCALLBACK {
			_, err = d.appCallbackClient.OnInvoke(context.Background(), d.invokeAppCallbackRequest)
		}
	} else if c == CAPABILITY_STATE {
		if t == TARGET_DAPR {
			_, err = d.daprClient.GetState(context.Background(), d.getStateRequest)
		}
	} else if c == CAPABILITY_PUBSUB {
		if t == TARGET_DAPR {
			switch m {
			case "publish":
				_, err = d.daprClient.PublishEvent(context.Background(), d.publishEventRequest)
			case "publish-multi":
				err = nil
				for _, req := range d.publishEventRequests {
					_, ierr := d.daprClient.PublishEvent(context.Background(), req)
					if ierr != nil {
						err = ierr
					}
				}
			case "bulkpublish":
				_, err = d.daprClient.BulkPublishEventAlpha1(context.Background(), d.bulkPublishRequest)
			}
		}
	}

	return err
}

func (d *DaprGRPCRunnerResults) parseDaprParameters(params string) error {
	d.params = &DaprRequestParameters{extensions: make(map[string]string)}

	kvs := strings.Split(params, ",")
	for _, kv := range kvs {
		kv := strings.Split(kv, "=")
		k := strings.TrimSpace(kv[0])
		v := strings.TrimSpace(kv[1])
		switch k {
		case "capability":
			d.params.capability = v
		case "target":
			d.params.target = v
		case "method":
			d.params.method = v
		case "appid":
			d.params.appId = v
		case "store":
			d.params.store = v
		default:
			d.params.extensions[k] = v
		}
	}

	return nil
}

// stringToMap converts a string of the form "key1=value1,key2=value2" to a map.
func stringToMap(s string) map[string]string {
	m := make(map[string]string)
	kvs := strings.Split(s, ",")
	for _, kv := range kvs {
		kv := strings.Split(kv, "=")
		if len(kv) != 2 {
			// ignore invalid key-value pair
			continue
		}
		k := strings.TrimSpace(kv[0])
		if k == "" {
			// ignore empty key
			continue
		}
		v := strings.TrimSpace(kv[1])
		if v == "" {
			// ignore empty value
			continue
		}
		m[k] = v
	}
	return m
}
