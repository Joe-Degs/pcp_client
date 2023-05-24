# PgPool-II Communication Protocol Client
A client that talks to a designated pcp backend to collect data (and possibly control the server).
This client provides a REST API for interacting with it.

## Build
```sh
go build .
```

## Run
```sh
./pcp_client --help
```
```
Usage of ./pcp_client:
  -client.addr string
        address for this pcp client (default "localhost:8080")
  -pcp.addr string
        address of the pcp backend (default "pool:9898")
  -pcp.password string
        password to use for authorization (default "password")
  -pcp.username string
        username of user to use for authorization (default "pgpool")

```