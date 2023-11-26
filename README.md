# disysBiddingReplication



protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative grpc/proto.proto

If you have issues when crashing a primary replicator (error message along the lines of "No connection could be made because the target machine actively refused it.") please turn off firewall
Simply accepting the program through when launching is not enough, it needs to be properly disabled