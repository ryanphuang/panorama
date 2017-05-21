package client

import (
	"net/rpc"
	"runtime"
	"sync"
	"time"

	du "deephealth/util"
)

const (
	tag        = "client"
	MaxRetries = 3
)

type RpcClient interface {
	Call(serviceMethod string, args interface{}, reply interface{}) error
}

type SimpleRpcClient struct {
	Addr string // server address to connect to
}

type PersistentRpcClient struct {
	Addr string // server address to connect to

	conn *rpc.Client // persistent RPC connection
	mu   *sync.Mutex // mutex
}

func (c *SimpleRpcClient) Call(serviceMethod string, args interface{}, reply interface{}) error {
	conn, err := rpc.Dial("tcp", c.Addr)
	if err != nil {
		return err
	}
	defer conn.Close()
	return conn.Call(serviceMethod, args, reply)
}

func (c *PersistentRpcClient) Connect() error {
	if c.conn != nil {
		return nil
	}
	var err error
	c.conn, err = rpc.Dial("tcp", c.Addr)
	return err
}

// Reconnect logic
func (c *PersistentRpcClient) Reconnect(maxretries int) error {
	var err error
	sleep := time.Duration(1)
	// retry for at most MaxRetries times with exponential back-off
	for retries := 1; retries <= maxretries; retries++ {
		du.LogI(tag, "(%s) server shut down, trying to reconnect, %d time(s)...", c.Addr, retries)
		c.Close()
		err = c.Connect()
		if err == nil {
			du.LogI(tag, "(%s) server is back online", c.Addr)
			break
		}
		// if it's the last retry and we haven't gone through yet,
		// there is no point to sleep...take a break!
		if retries != maxretries {
			du.LogD(tag, "(%s) sleeping for %d second(s)", c.Addr, sleep)
			time.Sleep(sleep * time.Second)
			sleep = sleep * 2
		}
	}
	return err
}

func (c *PersistentRpcClient) Call(serviceMethod string, args interface{}, reply interface{}) error {
	var err error
	// Mutex the connection in case there are concurrent RPC calls that cause race condition
	// to the connection
	c.mu.Lock()
	defer c.mu.Unlock()
	err = c.Connect() // each time we may re-use a connection or re-establish a connection
	if err != nil {
		return err
	}

	err = c.conn.Call(serviceMethod, args, reply)

	// the server is down :(
	if err == rpc.ErrShutdown {
		// we wont' do any failure handling here, but simply set the connection to nil
		// so next RPC call will reconnect, and for this RPC call we just return error
		c.Close() // close the shutdown connection

		if MaxRetries > 0 {
			err = c.Reconnect(MaxRetries)
			if err == nil && c.conn != nil {
				// connection back online, re-issue the RPC!
				err = c.conn.Call(serviceMethod, args, reply)
			} else if err != nil {
				err = rpc.ErrShutdown // force the return error to be rpc.ErrShutdown
				du.LogD(tag, "(%s) rpc server shutdown", c.Addr)
			}
		}
	}
	return err
}

// Tear down the RPC client
func (c *PersistentRpcClient) Close() {
	if c.conn != nil {
		c.conn.Close() // Release the persistent connection
		c.conn = nil   // Set to nil
	}
}

func finalizer(c *PersistentRpcClient) {
	c.Close()
}

var _ RpcClient = new(SimpleRpcClient)
var _ RpcClient = new(PersistentRpcClient)

// Creates an RPC client that connects to addr.
func NewPersistentRpcClient(addr string) *PersistentRpcClient {
	// We don't set the connection field until our first RPC call
	c := &PersistentRpcClient{
		Addr: addr,
		mu:   &sync.Mutex{},
	}
	// Set the finalizer so the persistent connection can be closed when client is GCed
	runtime.SetFinalizer(c, finalizer)
	return c
}

func NewSimpleRpcClient(addr string) *SimpleRpcClient {
	return &SimpleRpcClient{Addr: addr}
}
