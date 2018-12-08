# *Panorama*: Capturing and Enhancing In Situ System Observability for Failure Detection

This is the repository for the Panorama system described in our [OSDI '18](https://www.cs.jhu.edu/~huang/paper/panorama-osdi18.pdf) paper.

## Usage
### Requirements
Panorama is written in Go. To use it, you must have the Go compiler installed.
You can download the Go distribution from the [official website](https://golang.org/).
The code is tested with Go 1.8.

In addition, the RPC layer of Panorama is built on top of [gRPC](https://grpc.io)
and [Protocol Buffers](https://developers.google.com/protocol-buffers). A protobuf 
[compiler](https://github.com/protocolbuffers/protobuf/releases) is required. 
We have tested with `protoc` version 3.5.1 but recent versions should also work.
To parse the Panorama service definitions, the protobuf Go plugin `protoc-gen-go`
is also needed. We recommend to get the version [1.2.0](https://github.com/golang/protobuf/tree/v1.2.0), 
which supports Go 1.8. Both `protoc` and `protoc-gen-go` should be in the PATH.

### Download & Installation
Easiest way is to use `go get -u github.com/ryanphuang/panorama`. You can also
manually clone the repo to your `$GOPATH/src` and then build it with:
```
$ cd panorama
$ make
```

If you don't have the protobuf Go plugin, the Makefile provides a target to 
install it:
```
$ make tool-deps
```
Note that it will also install [dep](https://github.com/golang/dep) for 
dependency management.

Afterwards, you will find `hview-server`, `hview-client` in `$GOPATH/bin`.
Panorama also comes with a thin Java client wrapper library. To get the 
Java client library, type `make java`. The library will be generated
in `client/java/target/dh-client-1.0-SNAPSHOT-jar-with-dependencies.jar`.

## Generate Panorama service config

`$ hview-mkrc -fix_port 6688 -nserver 10 -addressp razor%d -namep pano%d -id pano0 -filter -subjects nn1,dn1,dn2,dn3`
will make a Panorama service config that consists of 10 Panorama instances `pano[0-9]`, each listening
to the address `razorX:6688`. This particular Panorama instance is identified by `pano0`.
The service is also configured to filter observations based on subjects, meaning only 
observations about `nn1,dn1,dn2,dn3` are accepted into the LOS. The resulted config is 
written to standard output:
```
{
    "Addr": "razor0:6688",
    "Id": "pano0",
    "Subjects": [
        "nn1",
        "dn1",
        "dn2",
        "dn3"
    ],
    "Peers": {
        "pano0": "razor0:6688",
        "pano1": "razor1:6688",
        "pano2": "razor2:6688",
        "pano3": "razor3:6688",
        "pano4": "razor4:6688",
        "pano5": "razor5:6688",
        "pano6": "razor6:6688",
        "pano7": "razor7:6688",
        "pano8": "razor8:6688",
        "pano9": "razor9:6688"
    },
    "FilterSubmission": true,
    "LogLevel": "",
    "DumpMemUsage": false,
    "DBFile": "deephealth.db",
}
```

`$ hview-mkrc -fix_port 6688 -nserver 10 -addressp razor%d -namep pano%d -id pano0 -output hs.cfg` will save the
configuration to file `hs.cfg` and prints its content to standard output.

## Starting Panorama instance

To start a single Panorama server, run the following command (replace `razor0` with
the hostname or just `localhost`),

`$ hview-server -addr razor0:6688 pano0`

Note that the service will run in the foreground. Support for a daemon service is in the TODO list. 
But a simple way is just to start with `hview-server -addr razor0 pano0 > dhs.log 2>&1 &`

To start a Panorama service with multiple participating peers, use the configuration
file generated above:

`$ hview-server -config hs.cfg`

## Using the log monitor tool to participate in observation reporting
For example, to use the ZooKeeper plugin of the logtail tool, run
`$ hview-logtail -stale=-1 -server razor0:6688 -log ~/software/zookeeper/zookeeper.out zookeeper --ensemble ~/software/zookeeper/conf/zoo.cfg  --filter conf/zoo_filter.json`

## Querying or reporting using Panorama client

### Submit an observation
To start Panorama client in an interactive mode, run

`$ hview-client -server razor0:6688`

The `-server razor0:6688` can be omitted if you are querying a local Panorama instance
listening at the default port.

An example session is as follows:

```bash
> help
Command list:
         me observer
         report subject [<metric:status:score...>]
         get [report|view|inference|panorama] [observer] subject 
         list [subject]
         dump [inference|panorama]
         ping
         help
         exit

> me peer@2
> report peer@1 snapshot:u:30
Accepted
> get report peer@1
observer:"peer@2" subject:"peer@1" observation:<ts:<seconds:1495181595 nanos:385767379 > metrics:<key:"snapshot" value:<name:"snapshot" value:<status:UNHEALTHY score:30 > > > >
>
```

On the server side, we can tail the server log `dhs.log`, which if successful may
produce something like this:

```
2017-05-19T08:13:15Z[DEBUG] raw.go:89: add report for peer@1 from peer@2...
2017-05-19T08:13:15Z[DEBUG] raw.go:114: create view for peer@2->peer@1...
2017-05-19T08:13:15Z[DEBUG] raw.go:117: add observation to view peer@2->peer@1: 2017-05-19T08:13:15.385767379Z { snapshot: UNHEALTHY, 30.0; }
2017-05-19T08:13:15Z[DEBUG] service.go:173: sent report for peer@1 for inference
2017-05-19T08:13:15Z[DEBUG] inference.go:82: received report for peer@1 for inference
2017-05-19T08:13:15Z[DEBUG] majority.go:55: score sum for snap is 30.000000
2017-05-19T08:13:15Z[DEBUG] inference.go:60: inference result for peer@1: 2017-05-19T08:13:15.387037413Z { snapshot: UNHEALTHY, 30.0; }
```

### Query health report

To list all subjects that have been observed,

```bash
$ hview-client list subject

peer@1  2017-05-21 08:00:39.367133633 +0000 UTC
peer@4  2017-05-21 08:00:39.35836465 +0000 UTC
peer@3  2017-05-21 08:00:39.360098717 +0000 UTC
peer@2  2017-05-21 08:00:39.361055495 +0000 UTC
peer@8  2017-05-21 08:00:39.362379457 +0000 UTC
peer@9  2017-05-21 08:00:39.365596665 +0000 UTC
```

To get a panorama for a particular subject,

```bash
$ hview-client get panorama peer@9

[[... peer@9->peer@9 (1 observations) ...]]
  |peer@9| 2017-05-19T17:16:25Z { SyncThread: UNHEALTHY, 20.0; }
```

To dump all inference for all observed subjects,

```bash
$ hview-client dump inference

=============peer@1=============
[peer@9] ==> peer@1: 2017-05-21T08:00:39.367278005Z { RecvWorker: UNHEALTHY, 20.0; }
=============peer@4=============
[peer@9] ==> peer@4: 2017-05-21T08:00:39.358928732Z { RecvWorker: UNHEALTHY, 20.0; }
=============peer@3=============
[peer@9] ==> peer@3: 2017-05-21T08:00:39.360242189Z { SendWorker: UNHEALTHY, 20.0; }
=============peer@2=============
[peer@9] ==> peer@2: 2017-05-21T08:00:39.361172754Z { SendWorker: UNHEALTHY, 20.0; }
=============peer@8=============
[peer@9] ==> peer@8: 2017-05-21T08:00:39.362531433Z { SendWorker: UNHEALTHY, 20.0; }
=============peer@9=============
[peer@9] ==> peer@9: 2017-05-21T08:00:39.365718626Z { SyncThread: UNHEALTHY, 20.0; }
```

## TODO

- [x] Parallelize report propagation
- [ ] Re-initialize state from report db after restart
