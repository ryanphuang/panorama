package client

import (
	pb "deephealth/build/gen"
)

type NClient struct {
	r RpcClient
}

func NewClient(addr string, persistent bool) *NClient {
	if persistent {
		r := NewPersistentRpcClient(addr)
		return &NClient{r: r}
	} else {
		r := NewSimpleRpcClient(addr)
		return &NClient{r: r}
	}
}

func (c *NClient) ObserveSubject(subject string, reply *bool) error {
	return c.r.Call("HealthNServer.ObserveSubject", subject, &reply)
}

func (c *NClient) StopObservingSubject(subject string, reply *bool) error {
	return c.r.Call("HealthNServer.StopObservingSubject", subject, &reply)
}

func (c *NClient) SubmitReport(report *pb.Report, reply *int) error {
	return c.r.Call("HealthNServer.SubmitReport", report, &reply)
}

func (c *NClient) GossipReport(report *pb.Report, reply *int) error {
	return c.r.Call("HealthNServer.GossipReport", report, &reply)
}

func (c *NClient) GetLatestReport(subject string, reply *pb.Report) error {
	return c.r.Call("HealthNServer.GetLatestReport", subject, &reply)
}
