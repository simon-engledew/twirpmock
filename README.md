# twirpmock

Mock out any server that speaks [Twirp](https://github.com/twitchtv/twirp) (a simple RPC framework built on protobuf).

All you need is the protobuf definition of your service, e.g:

```proto
...
service Example {
  rpc Echo(EchoRequest) returns (EchoResponse);
}

message EchoRequest {
  string name = 1;
}

message EchoResponse {
  string message = 1;
  google.protobuf.Timestamp timestamp = 2;
}
```

and a script for the fake service to follow when any of the endpoints are called. This is written in [Starlark](https://github.com/bazelbuild/starlark), the Bazel configuration language:

```python
def Echo(request, response):
    response.message = "Hello " + request.name + "!"
    response.timestamp = now()
    return response
```

Now when a client connects twirpmock will follow the script to create a response!

The `generate` function from [gofakeit](https://github.com/brianvoe/gofakeit) is also available to allow you to create a range of fake data:

```python
def Echo(request, response):
    response.message = generate('here is a uuid {uuid} for you ') + request.name + '!'
    response.timestamp = now()
    return response
```

Twirpmock can be run standalone or as a Docker container:

```sh
docker run \
  -p 8888:8888 \
  -v "$(pwd):/workspace" \
  -w /workspace \
  ghcr.io/simon-engledew/twirpmock:latest example/service.proto.pb example/service.star
```
