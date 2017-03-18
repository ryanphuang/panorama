package client

import (
	dt "deephealth/types"
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

func (c *NClient) ObserveSubject(subject dt.EntityId, reply *bool) error {
	return c.r.Call("HealthNServer.ObserveSubject", subject, &reply)
}

func (c *NClient) StopObservingSubject(subject dt.EntityId, reply *bool) error {
	return c.r.Call("HealthNServer.StopObservingSubject", subject, &reply)
}

func (c *NClient) AddReport(report *dt.Report, reply *int) error {
	return c.r.Call("HealthNServer.AddReport", report, &reply)
}

func (c *NClient) GossipReport(report *dt.Report, reply *int) error {
	return c.r.Call("HealthNServer.GossipReport", report, &reply)
}

func (c *NClient) GetReport(subject dt.EntityId, reply *dt.Report) error {
	return c.r.Call("HealthNServer.GetReport", subject, &reply)
}
