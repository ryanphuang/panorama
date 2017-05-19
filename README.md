# Deep Health Check to Detect Gray Failure

## Generate health server config

`$ hview-mkrc -fix_port 6688 -nserver 10 -addressp pano%d -namep HS_%d -id HS_1`

produces

```
{
    "Addr": "pano0:6688",
    "Id": "HS_1",
    "Subjects": null,
    "Peers": {
        "HS_1": "pano0:6688",
        "HS_10": "pano9:6688",
        "HS_2": "pano1:6688",
        "HS_3": "pano2:6688",
        "HS_4": "pano3:6688",
        "HS_5": "pano4:6688",
        "HS_6": "pano5:6688",
        "HS_7": "pano6:6688",
        "HS_8": "pano7:6688",
        "HS_9": "pano8:6688"
    },
    "FilterSubmission": false
}
```

## Starting health server

`$ hview-server -addr instance1 -grpc DHS_1`

## Starting health client

`$ hview-client -grpc instance1:15045`

## Monitor log
`$ hview-logtail instance3:6688 ~/software/zookeeper/zookeeper.out ensemble.cfg`
