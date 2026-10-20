package resources

import (
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
)

// apiGatewayCallLog records the API Gateway calls a mock receives. The nukers delete concurrently, so it locks.
type apiGatewayCallLog struct {
	mu    sync.Mutex
	calls []string
}

func (l *apiGatewayCallLog) add(call string) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.calls = append(l.calls, call)
}

func (l *apiGatewayCallLog) list() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]string(nil), l.calls...)
}

// apiGatewayPage maps a mock page token ("1", "2") to the index of the page it names; no token is the first page.
// API Gateway v2 calls the token NextToken and API Gateway v1 calls it Position.
func apiGatewayPage(token *string) int {
	switch aws.ToString(token) {
	case "":
		return 0
	case "1":
		return 1
	case "2":
		return 2
	}
	panic("unexpected page token " + aws.ToString(token))
}
