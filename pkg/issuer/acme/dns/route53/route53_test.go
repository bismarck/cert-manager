package route53

import (
	"net/http/httptest"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/stretchr/testify/assert"
)

var (
	route53Secret string
	route53Key    string
	route53Region string
	route53Zone   string
)

func init() {
	route53Key = os.Getenv("AWS_ACCESS_KEY_ID")
	route53Secret = os.Getenv("AWS_SECRET_ACCESS_KEY")
	route53Region = os.Getenv("AWS_REGION")
}

func restoreRoute53Env() {
	os.Setenv("AWS_ACCESS_KEY_ID", route53Key)
	os.Setenv("AWS_SECRET_ACCESS_KEY", route53Secret)
	os.Setenv("AWS_REGION", route53Region)
}

func makeRoute53Provider(ts *httptest.Server) *DNSProvider {
	config := &aws.Config{
		Credentials: credentials.NewStaticCredentials("abc", "123", " "),
		Endpoint:    aws.String(ts.URL),
		Region:      aws.String("mock-region"),
		MaxRetries:  aws.Int(1),
	}

	client := route53.New(session.New(config))
	return &DNSProvider{client: client}
}

func TestAmbientCredentialsFromEnv(t *testing.T) {
	os.Setenv("AWS_ACCESS_KEY_ID", "123")
	os.Setenv("AWS_SECRET_ACCESS_KEY", "123")
	os.Setenv("AWS_REGION", "us-east-1")
	defer restoreRoute53Env()

	provider, err := NewDNSProvider("", "", "", "", true)
	assert.NoError(t, err, "Expected no error constructing DNSProvider")

	_, err = provider.client.Config.Credentials.Get()
	assert.NoError(t, err, "Expected credentials to be set from environment")
}

func TestNoCredentialsFromEnv(t *testing.T) {
	os.Setenv("AWS_ACCESS_KEY_ID", "123")
	os.Setenv("AWS_SECRET_ACCESS_KEY", "123")
	os.Setenv("AWS_REGION", "us-east-1")
	defer restoreRoute53Env()

	_, err := NewDNSProvider("", "", "", "", false)
	assert.Error(t, err, "Expected error constructing DNSProvider with no credentials and not ambient")
}

func TestAmbientRegionFromEnv(t *testing.T) {
	os.Setenv("AWS_REGION", "us-east-1")
	defer restoreRoute53Env()

	provider, err := NewDNSProvider("", "", "", "", true)
	assert.NoError(t, err, "Expected no error constructing DNSProvider")

	assert.Equal(t, "us-east-1", *provider.client.Config.Region, "Expected Region to be set from environment")
}

func TestRoute53Present(t *testing.T) {
	mockResponses := MockResponseMap{
		"/2013-04-01/hostedzonesbyname":         MockResponse{StatusCode: 200, Body: ListHostedZonesByNameResponse},
		"/2013-04-01/hostedzone/ABCDEFG/rrset/": MockResponse{StatusCode: 200, Body: ChangeResourceRecordSetsResponse},
		"/2013-04-01/change/123456":             MockResponse{StatusCode: 200, Body: GetChangeResponse},
	}

	ts := newMockServer(t, mockResponses)
	defer ts.Close()

	provider := makeRoute53Provider(ts)

	domain := "example.com"
	keyAuth := "123456d=="

	err := provider.Present(domain, "", keyAuth)
	assert.NoError(t, err, "Expected Present to return no error")
}
