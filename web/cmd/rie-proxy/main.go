// Command rie-proxy translates plain HTTP requests into Lambda Function URL
// (API Gateway HTTP API v2) invoke payloads for the AWS Lambda Runtime
// Interface Emulator, and translates the responses back. It lets a browser
// talk to the exact same Lambda binary that runs in production.
package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/aws/aws-lambda-go/events"
)

func main() {
	target := os.Getenv("LAMBDA_RIE_URL")
	if target == "" {
		target = "http://web:8080/2015-03-31/functions/function/invocations"
	}
	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8080"
	}

	client := &http.Client{Timeout: 30 * time.Second}

	// The Lambda Runtime Interface Emulator processes one /invocations
	// request at a time; a concurrent second request crashes it
	// (rapidcore "AlreadyReserved" -> nil pointer panic). Serialize.
	var mu sync.Mutex

	log.Printf("rie-proxy listening on %s, forwarding to %s", addr, target)
	log.Fatal(http.ListenAndServe(addr, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		proxy(w, r, client, target)
	})))
}

func proxy(w http.ResponseWriter, r *http.Request, client *http.Client, target string) {
	event, err := toEvent(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	payload, err := json.Marshal(event)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp, err := client.Post(target, "application/json", bytes.NewReader(payload))
	if err != nil {
		http.Error(w, "lambda invoke failed: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	if resp.Header.Get("X-Amz-Function-Error") != "" {
		log.Printf("lambda function error: %s", body)
		http.Error(w, "lambda function error: "+string(body), http.StatusBadGateway)
		return
	}

	var out events.APIGatewayV2HTTPResponse
	if err := json.Unmarshal(body, &out); err != nil {
		http.Error(w, "invalid lambda response: "+err.Error(), http.StatusBadGateway)
		return
	}

	writeResponse(w, out)
}

func toEvent(r *http.Request) (events.APIGatewayV2HTTPRequest, error) {
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		return events.APIGatewayV2HTTPRequest{}, err
	}

	body := string(bodyBytes)
	isBase64 := false
	if !utf8.Valid(bodyBytes) {
		body = base64.StdEncoding.EncodeToString(bodyBytes)
		isBase64 = true
	}

	headers := make(map[string]string, len(r.Header))
	var cookies []string
	for k, v := range r.Header {
		if strings.EqualFold(k, "Cookie") {
			for _, c := range v {
				for _, part := range strings.Split(c, ";") {
					part = strings.TrimSpace(part)
					if part != "" {
						cookies = append(cookies, part)
					}
				}
			}
			continue
		}
		headers[strings.ToLower(k)] = strings.Join(v, ",")
	}

	sourceIP := r.RemoteAddr
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		sourceIP = host
	}

	return events.APIGatewayV2HTTPRequest{
		Version:         "2.0",
		RouteKey:        "$default",
		RawPath:         r.URL.EscapedPath(),
		RawQueryString:  r.URL.RawQuery,
		Cookies:         cookies,
		Headers:         headers,
		Body:            body,
		IsBase64Encoded: isBase64,
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			DomainName: r.Host,
			HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
				Method:    r.Method,
				Path:      r.URL.EscapedPath(),
				Protocol:  r.Proto,
				SourceIP:  sourceIP,
				UserAgent: r.UserAgent(),
			},
		},
	}, nil
}

func writeResponse(w http.ResponseWriter, out events.APIGatewayV2HTTPResponse) {
	h := w.Header()
	for k, v := range out.Headers {
		h.Set(k, v)
	}
	for _, c := range out.Cookies {
		h.Add("Set-Cookie", c)
	}

	status := out.StatusCode
	if status == 0 {
		status = http.StatusOK
	}
	w.WriteHeader(status)

	if out.Body == "" {
		return
	}

	if out.IsBase64Encoded {
		decoded, err := base64.StdEncoding.DecodeString(out.Body)
		if err != nil {
			log.Printf("failed to decode base64 body: %v", err)
			return
		}
		w.Write(decoded)
		return
	}

	io.WriteString(w, out.Body)
}
