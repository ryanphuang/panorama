package client

import (
	dh "deephealth"
)

type Client struct {
	r RpcClient
}

var _ dh.HealthService = new(Client)

func NewClient(addr string, persistent bool) *Client {
	if persistent {
		r := NewPersistentRpcClient(addr)
		return &Client{r: r}
	} else {
		r := NewSimpleRpcClient(addr)
		return &Client{r: r}
	}
}

func (c *Client) ObserveSubject(subject dh.EntityId, reply *bool) error {
	return c.r.Call("HealthService.ObserveSubject", subject, &reply)
}

func (c *Client) StopObservingSubject(subject dh.EntityId, reply *bool) error {
	return c.r.Call("HealthService.StopObservingSubject", subject, &reply)
}

func (c *Client) AddReport(report *dh.Report, reply *int) error {
	return c.r.Call("HealthService.AddReport", report, &reply)
}

func (c *Client) GossipReport(report *dh.Report, reply *int) error {
	return c.r.Call("HealthService.GossipReport", report, &reply)
}

func (c *Client) GetReport(subject dh.EntityId, reply *dh.Report) error {
	return c.r.Call("HealthService.GetReport", subject, &reply)
}
