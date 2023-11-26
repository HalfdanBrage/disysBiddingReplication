# BiddingReplication
Made by tpep, dhla and habr

## Running the program

All of the following commands are meant to be called from the project´s root directory.\
Before running the commands you should decide on a port to use (eg. 6543).\
All commands must use the same port to be considered the same "auction session."\
Both clients and secondary replicators rely on the existance of a primary replicator on the selected port.\

### Server


#### Start a primary replicator
    go run server/replicator.go --role primary --time 60 --port xxxx
Time represents the amount of time the auction should take in seconds

#### Start a secondary replicator
    go run server/replicator.go --port xxxx
Secondary replicators are reliant on a primary replicator to already exists on the selected port.\
Therefore start a primary replicator before attempting to start a secondary.

### Client

#### Start a client
    go run client/client.go --port xxxx
Clients are also reliant on a primary replicator existing on startup.\
When starting the client you will be prompted to input a name. This name is used to identify the bidder, so don't use the same name for multiple different bidders


## Known fixes to issues

If you have issues when crashing a primary replicator (error message along the lines of "No connection could be made because the target machine actively refused it.") please turn off your firewall.
Simply accepting the program through the firewall when launching does not always work (more specifically it isn't enough if you aren't prompted to let it through the firewall whenever a secondary replicator attempts to reconnect as primary)

When proto file is updated use: 
protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative grpc/proto.proto
