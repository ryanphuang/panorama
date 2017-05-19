# Deep and Collaborative Health Check to Detect Gray Failure

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

## Starting deep health client

To start an interactive deep health client, run

`$ hview-client pano0:6688`

An example session is as follows:

```bash
> help
Command list:
         me observer
         report subject [<metric:status:score...>]
         get [report|view|panorama] [observer] subject
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

> 2017-05-19T08:13:15Z[DEBUG] raw.go:89: add report for TS_2 from TS_1...
> 2017-05-19T08:13:15Z[DEBUG] raw.go:114: create view for TS_1->TS_2...
> 2017-05-19T08:13:15Z[DEBUG] raw.go:117: add observation to view TS_1->TS_2: 2017-05-19T08:13:15.385767379Z { snapshot: UNHEALTHY, 30.0; }
> 2017-05-19T08:13:15Z[DEBUG] service.go:173: sent report for TS_2 for inference
> 2017-05-19T08:13:15Z[DEBUG] inference.go:82: received report for TS_2 for inference
> 2017-05-19T08:13:15Z[DEBUG] majority.go:55: score sum for snap is 30.000000
> 2017-05-19T08:13:15Z[DEBUG] inference.go:60: inference result for TS_2: 2017-05-19T08:13:15.387037413Z { snapshot: UNHEALTHY, 30.0; }

## Using the log monitor tool to participate in deep health reporting
`$ hview-logtail pano3:6688 ~/software/zookeeper/zookeeper.out ensemble.cfg`
