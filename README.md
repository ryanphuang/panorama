# *Panorama*: Capturing and Enhancing In Situ System Observability for Failure Detection

## Usage
### Requirements
Panorama is written in Go. To use it, you must have the Go compiler installed.
You can download the Go distribution from the [official website](https://golang.org/).

In addition, the RPC layer of Panorama is built on top of [gRPC](https://grpc.io) 
and [Protocol Buffers](https://developers.google.com/protocol-buffers). You 
can get the protobuf for Go with `go get -u github.com/golang/protobuf/protoc-gen-go`.
This will install the protobuf compiler at `$GOPATH/bin`. **Note**: The recent 
Go protoc3 introduces [changes](https://groups.google.com/forum/#!topic/protobuf/N-elvFu4dFM) 
that will generate additional fields such as `XXX_NoUnkeyedLiteral` in 
the protobuf messages, which may cause side effects when using the message 
in map keys or `==` test. We haven't double checked Panorama's code to 
eliminate such side effects yet.

### Installation
Clone the source tree of Panorama to your `$GOPATH/src`, then make it:
```
$ cd $GOPATH
$ git clone git@github.com:ryanphuang/deephealth.git
$ make
```

Afterwards, you will find `hview-server`, `hview-client` in `$GOPATH/bin`.
Panorama also comes with a thin Java client wrapper library. To get the 
Java client library, type `make java`. The library will be generated
in `client/java/target/dh-client-1.0-SNAPSHOT-jar-with-dependencies.jar`.

## Generate deep health server config

`$ hview-mkrc -fix_port 6688 -nserver 10 -addressp pano%d -namep DHS_%d -id DHS_1 -filter -subject TS_1,TS_2,TS_3`

produces to standard output

```
{
    "Addr": "pano0:6688",
    "Id": "DHS_1",
    "Subjects": [
        "TS_1",
        "TS_2",
        "TS_3"
    ],
    "Peers": {
        "DHS_1": "pano0:6688",
        "DHS_10": "pano9:6688",
        "DHS_2": "pano1:6688",
        "DHS_3": "pano2:6688",
        "DHS_4": "pano3:6688",
        "DHS_5": "pano4:6688",
        "DHS_6": "pano5:6688",
        "DHS_7": "pano6:6688",
        "DHS_8": "pano7:6688",
        "DHS_9": "pano8:6688"
    },
    "FilterSubmission": true
}
```

`$ hview-mkrc -fix_port 6688 -nserver 10 -addressp pano%d -namep DHS_%d -id DHS_1 -output hs.cfg` will save the
configuration to file `hs.cfg` and prints its content to standard output.

## Starting deep health server

To start a single deep health server, run the following command (replace `pano0` with
the hostname or just `localhost`),

`$ hview-server -addr pano0:6688 DHS_1`

Note that the service will run in the foreground. Support for a daemon service is in the TODO list. 
But a simple way is just to start with `hview-server -addr pano0 DHS_1 > dhs.log 2>&1 &`

To start a deep health service with multiple participating peers, use the configuration
file generated before:

`$ hview-server -config hs.cfg`

## Using the log monitor tool to participate in deep health reporting
For example, to use the ZooKeeper plugin of the logtail tool, run
`$ hview-logtail -stale=-1 -server pano0:6688 -log ~/software/zookeeper/zookeeper.out zookeeper --ensemble ~/software/zookeeper/conf/zoo.cfg  --filter conf/zoo_filter.json`

## Querying or reporting using deep health client

### Submit a health report
To start deep health client in an interactive mode, run

`$ hview-client -server pano0:6688`

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

> me TS_1
> report TS_2 snapshot:u:30
Accepted
> get report TS_2
observer:"TS_1" subject:"TS_2" observation:<ts:<seconds:1495181595 nanos:385767379 > metrics:<key:"snapshot" value:<name:"snapshot" value:<status:UNHEALTHY score:30 > > > >
>
```

On the server side, we can tail the server log `dhs.log`, which if successful may
produce something like this:

```
2017-05-19T08:13:15Z[DEBUG] raw.go:89: add report for TS_2 from TS_1...
2017-05-19T08:13:15Z[DEBUG] raw.go:114: create view for TS_1->TS_2...
2017-05-19T08:13:15Z[DEBUG] raw.go:117: add observation to view TS_1->TS_2: 2017-05-19T08:13:15.385767379Z { snapshot: UNHEALTHY, 30.0; }
2017-05-19T08:13:15Z[DEBUG] service.go:173: sent report for TS_2 for inference
2017-05-19T08:13:15Z[DEBUG] inference.go:82: received report for TS_2 for inference
2017-05-19T08:13:15Z[DEBUG] majority.go:55: score sum for snap is 30.000000
2017-05-19T08:13:15Z[DEBUG] inference.go:60: inference result for TS_2: 2017-05-19T08:13:15.387037413Z { snapshot: UNHEALTHY, 30.0; }
```

### Query health report

To list all subjects that have been observed,

```bash
$ hview-client -server pano0:6688 list subject

peer@1  2017-05-21 08:00:39.367133633 +0000 UTC
peer@4  2017-05-21 08:00:39.35836465 +0000 UTC
peer@3  2017-05-21 08:00:39.360098717 +0000 UTC
peer@2  2017-05-21 08:00:39.361055495 +0000 UTC
peer@8  2017-05-21 08:00:39.362379457 +0000 UTC
peer@9  2017-05-21 08:00:39.365596665 +0000 UTC
```

To get a panorama for a particular subject,

```bash
$ hview-client -server pano0:6688 get panorama peer@9

[[... peer@9->peer@9 (1 observations) ...]]
  |peer@9| 2017-05-19T17:16:25Z { SyncThread: UNHEALTHY, 20.0; }
```

To dump all inference for all observed subjects,

```bash
$ hview-client -server pano0:6688 dump inference

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

