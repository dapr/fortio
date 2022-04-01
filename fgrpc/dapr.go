package fgrpc

import (
	"context"
	v1 "fortio.org/fortio/dapr/proto/common/v1"
	dapr "fortio.org/fortio/dapr/proto/runtime/v1"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/anypb"
)

type DaprGRPCRunnerResults struct {
	client dapr.DaprClient

	invokeRequest *dapr.InvokeServiceRequest
}

func (d *DaprGRPCRunnerResults) NewClient(conn *grpc.ClientConn) error {

	d.client = dapr.NewDaprClient(conn)
	return nil
}

func (d *DaprGRPCRunnerResults) PrepareRequest(o *GRPCRunnerOptions) {
	d.invokeRequest = &dapr.InvokeServiceRequest{
		Id: o.Service,
		Message: &v1.InvokeRequest{
			Method:      "loadTest", // hard code method name for load test
			ContentType: "text/plain",
		},
	}

	if len(o.Payload) > 0 {
		d.invokeRequest.Message.Data = &anypb.Any{Value: []byte(o.Payload)}
	} else {
		d.invokeRequest.Message.Data = &anypb.Any{Value: []byte{}}
	}
}

func (d *DaprGRPCRunnerResults) RunTest() error {
	_, error := d.client.InvokeService(context.Background(), d.invokeRequest)
	return error
}
