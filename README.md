# Go Api Gateway HTTP Adapter
This library is adapter between AWS Api Gateway event (payload version 2) and
standard Go HTTP interface. In concrete terms, it converts 
`github.com/aws/aws-lambda-go/events.APIGatewayV2HTTPRequest` to `*http.Request`
and content from `http.ResponseWriter` back to 
`github.com/aws/aws-lambda-go/events.APIGatewayV2HTTPResponse`, which is
suitable to be returned from Lambda to API Gateway.

There are number of similar solutions like this out there, but I could not find
one for Payload Format Version 2. 

## Example
At most basic level, this library exposes only one really important function: `Adapt`. 

It takes existing `http.Handler`, which would be invoked during processing of
Lambda event, and returns Lambda handler. Simple example might look like this:

```go
package server

import (
	"net/http"

	"github.com/aws/aws-lambda-go/lambda"
	adapter "github.com/delicb/go-apigateway-http-adapter"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/path", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello from lambda"))
	})
	lambda.Start(adapter.Adapt(mux))
}

```

Lambda handler returned by `Adapt` method has signature
`func(ctx context.Context, ev events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error))`, 
where `events` is `github.com/aws/aws-lambda-go/events`. It has not been tested or intended to 
work with non-V2 version of the payload. 

`*http.Request` instance provided to `http.Handler` will be populated with information 
obtained from API Gateway request and API Gateway response will be generated from
status, headers and body written to `http.ResponseWriter`. Note that `ResponseWriter` is
dummy object and will not implement things like `http.Pusher` or `http.CloseNotifier`. 

Additionally, original event can be obtained by using `APIGatewayRequest` method. For example, to get
`Authorizer` provided by API gateway withing HTTP handler, do the following:

```go
func handler(w http.ResponseWriter, r *http.Request) {
	ev, ok := adapter.APIGatewayRequest(r)
	if ok {
		// do something with authorizer
		// ev.RequestContext.Authorizer
	}
}
```

## Known issues
- [ ] Not everything is covered with tests and existing ones could use more love
- [ ] More advanced logic around base64 response body encoding is needed

## Contributing
Feel free to open tickets (or even better, pull requests). I am opened to suggestions. 

## Licence
Cliware is released under MIT licence.
