# Deep Health Check to Detect Gray Failure

## Starting health server

`$ hview-server -addr instance1 -grpc DHS_1`

## Starting health client

`$ hview-client -grpc instance1:15045`
