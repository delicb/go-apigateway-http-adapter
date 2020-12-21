package apigateway_adapter

import (
	"bytes"
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/aws/aws-lambda-go/events"
)

// Adapt returns lambda handler that passes processing to http.Handler.
//
// Adapt converts received API Gateway event to *http.Request instance, invokes
// handler and converts response to API Gateway response. It only works with
// version 2 API Gateway integration protocol.
//
// Example usage:
//   lambda.Start(Adapt(httpServer))
func Adapt(handler http.Handler) func(ctx context.Context, ev events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	return func(ctx context.Context, ev events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
		ctx = setApiGatewayEvent(ctx, ev)
		req, err := agw2httpRequest(ctx, ev)
		if err != nil {
			return events.APIGatewayV2HTTPResponse{}, err
		}
		proxyWriter := newProxyResponseWriter()
		handler.ServeHTTP(proxyWriter, req)
		return proxyWriter.toApiGatewayResponse(), nil
	}
}

func agw2httpRequest(ctx context.Context, ev events.APIGatewayV2HTTPRequest) (*http.Request, error) {
	// prepare the body
	var body io.Reader
	if ev.IsBase64Encoded {
		decodedBody, err := base64.StdEncoding.DecodeString(ev.Body)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(decodedBody)
	} else {
		body = strings.NewReader(ev.Body)
	}

	// prepare path
	path := ev.RawPath + "?" + ev.RawQueryString

	// create request
	req, err := http.NewRequestWithContext(
		ctx,
		strings.ToUpper(ev.RequestContext.HTTP.Method),
		path,
		body,
	)
	if err != nil {
		return nil, err
	}

	// populate some additional information for URL
	req.URL.Host = ev.RequestContext.DomainName
	req.Host = ev.RequestContext.DomainName
	if schema, ok := ev.Headers["x-forwarded-proto"]; ok {
		req.URL.Scheme = schema
	}

	// populate headers
	for h := range ev.Headers {
		// per Api Gateway documentation and https://www.w3.org/Protocols/rfc2616/rfc2616-sec4.html#sec4.2
		// it is possible to have multiple values in single header separated by comma.
		// Note that cookies are exception to this rule, so they are treated differently below
		parts := strings.Split(ev.Headers[h], ",")
		for _, part := range parts {
			req.Header.Add(h, strings.TrimSpace(part))
		}
	}

	// Cookies receive special treatment from Api Gateway
	for _, cookie := range ev.Cookies {
		req.Header.Add("Cookie", cookie)
	}

	return req, nil
}

type proxyResponseWriter struct {
	status  int
	headers http.Header
	body    *bytes.Buffer
}

func newProxyResponseWriter() *proxyResponseWriter {
	return &proxyResponseWriter{
		status:  0,
		headers: make(http.Header),
		body:    &bytes.Buffer{},
	}
}

func (w *proxyResponseWriter) Write(body []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.body.Write(body)
}

func (w *proxyResponseWriter) Header() http.Header {
	return w.headers
}

func (w *proxyResponseWriter) WriteHeader(status int) {
	w.status = status
}

func (w *proxyResponseWriter) toApiGatewayResponse() events.APIGatewayV2HTTPResponse {
	// single value headers
	headers := make(map[string]string)
	// multi value headers
	multiValueHeaders := make(map[string][]string)
	cookies := make([]string, 0)

	// extract headers
	for h := range w.headers {
		headerValues := w.headers.Values(h)

		// cookies have special treatment
		if strings.ToLower(h) == "set-cookie" {
			cookies = append(cookies, headerValues...)
			continue
		}

		// depending on number of values, we populate single or multi value headers
		if len(headerValues) == 1 {
			headers[h] = headerValues[0]
		} else {
			multiValueHeaders[h] = append(multiValueHeaders[h], headerValues...)
		}
	}

	// prepare body
	var body string
	var isBase64 bool

	rawBytes := w.body.Bytes()
	if utf8.Valid(rawBytes) {
		body = w.body.String()
		isBase64 = false
	} else {
		body = base64.StdEncoding.EncodeToString(rawBytes)
		isBase64 = true
	}

	return events.APIGatewayV2HTTPResponse{
		StatusCode:        w.status,
		Headers:           headers,
		MultiValueHeaders: multiValueHeaders,
		Body:              body,
		IsBase64Encoded:   isBase64,
		Cookies:           cookies,
	}
}

// Context support

// type for context key for storing api gateway event used to create
type gwEventKeyType string

// constant key for storing and extracting api gateway event in context
const gwEventKey gwEventKeyType = "gateway-event-key"

// APIGatewayRequest returns original APIGatewayV2HTTPRequest event that was
// used to create *http.Request instance. Second return parameter is flag indicating
// if event exists attached to the request. If it is false, returned APIGatewayV2HTTPRequest
// is empty value and should not be consumed.
func APIGatewayRequest(req *http.Request) (events.APIGatewayV2HTTPRequest, bool) {
	val, ok := req.Context().Value(gwEventKey).(events.APIGatewayV2HTTPRequest)
	return val, ok
}

func setApiGatewayEvent(ctx context.Context, ev events.APIGatewayV2HTTPRequest) context.Context {
	return context.WithValue(ctx, gwEventKey, ev)
}
