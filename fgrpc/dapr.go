package fgrpc

import (
	"context"
	"fmt"
	v1 "github.com/dapr/dapr/pkg/proto/common/v1"
	dapr "github.com/dapr/dapr/pkg/proto/runtime/v1"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/anypb"
	"strings"
)

const CAPABILITY_INVOKE = "invoke"
const TARGET_DAPR = "dapr"
const TARGET_APPCALLBACK = "appcallback"

type DaprGRPCRunnerResults struct {
	params                   *DaprRequestParameters
	daprClient               dapr.DaprClient
	daprInvokeRequest        *dapr.InvokeServiceRequest
	appCallbackClient        dapr.AppCallbackClient
	appCallbackInvokeRequest *v1.InvokeRequest
}

type DaprRequestParameters struct {
	capability string
	target     string
	method     string
	appId      string

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
	if t == TARGET_DAPR {
		d.daprClient = dapr.NewDaprClient(conn)
		if c == CAPABILITY_INVOKE {
			err = d.prepareRequest4InvokeDapr(o)
		}
	} else if t == TARGET_APPCALLBACK {
		d.appCallbackClient = dapr.NewAppCallbackClient(conn)
		if c == CAPABILITY_INVOKE {
			err = d.prepareRequest4InvokeAppCallback(o)
		}
	}

	return err
}

func (d *DaprGRPCRunnerResults) prepareRequest4InvokeDapr(o *GRPCRunnerOptions) error {
	method := d.params.method
	if method == "" {
		return fmt.Errorf("method is required for load test")
	}

	d.daprInvokeRequest = &dapr.InvokeServiceRequest{
		Id: d.params.appId,
		Message: &v1.InvokeRequest{
			Method:      method,
			ContentType: "text/plain",
		},
	}

	if len(o.Payload) > 0 {
		d.daprInvokeRequest.Message.Data = &anypb.Any{Value: []byte(o.Payload)}
	} else {
		d.daprInvokeRequest.Message.Data = &anypb.Any{Value: []byte{}}
	}
	return nil
}
func (d *DaprGRPCRunnerResults) prepareRequest4InvokeAppCallback(o *GRPCRunnerOptions) error {
	method := d.params.method
	if method == "" {
		return fmt.Errorf("method is required for load test")
	}

	d.appCallbackInvokeRequest = &v1.InvokeRequest{
		Method:      method,
		ContentType: "text/plain",
	}
	if len(o.Payload) > 0 {
		d.appCallbackInvokeRequest.Data = &anypb.Any{Value: []byte(o.Payload)}
	} else {
		d.appCallbackInvokeRequest.Data = &anypb.Any{Value: []byte{}}
	}
	return nil
}

func (d *DaprGRPCRunnerResults) RunTest() error {
	t := d.params.target
	c := d.params.capability
	err := fmt.Errorf("unsupported dapr load test: capability=%s, target=%s", c, t)

	if c == CAPABILITY_INVOKE {
		if t == TARGET_DAPR {
			_, err = d.daprClient.InvokeService(context.Background(), d.daprInvokeRequest)
		} else if t == TARGET_APPCALLBACK {
			_, err = d.appCallbackClient.OnInvoke(context.Background(), d.appCallbackInvokeRequest)
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
		default:
			d.params.extensions[k] = v
		}
	}

	return nil
}
