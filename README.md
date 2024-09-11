# fprc-poc

Simple POC of [`fprc-go`](https://github.com/loopholelabs/frpc-go/) two-way
stream communication.

## Quick start

Build the client and server binaries and start them in different terminals.

```
make
./out/server
./out/clent
```

Send sample requests.

```
curl http://localhost:8080/instances
curl -XPOST http://localhost:8080/instances
```
