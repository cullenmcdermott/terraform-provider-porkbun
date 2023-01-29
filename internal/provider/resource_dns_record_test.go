package provider

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/nrdcg/porkbun"
	"github.com/stretchr/testify/require"
)

type porkbunTestProvider struct {
	client     *porkbun.Client
	configured bool
	version    string
}

type Client struct {
	secretApiKey string
	apiKey       string
	BaseURL      *url.URL
	HTTPClient   *http.Client
}

type createResponse struct {
	Status string
	ID     int
}

type deleteResponse struct {
	Status string
}

type retrieveResponse struct {
	Status  string
	Records []porkbun.Record
}

func getProvider(client *porkbun.Client) porkbunProvider {
	return porkbunProvider{
		client: client,
	}
}

func newPorkbunProvider(testUrl string) provider.Provider {
	client := porkbun.New("sk1_foobarbaz", "pk1_foobarbaz")
	client.BaseURL, _ = url.Parse(testUrl)
	return &porkbunProvider{
		client:     client,
		configured: true,
		version:    "test",
	}
}

func TestPorkbunDnsRecordSuccess(t *testing.T) {
	testUrl, expectRequest, _ := MockPorkbun(t)
	os.Setenv("PORKBUN_API_KEY", "pk1_foobarbaz")
	os.Setenv("PORKBUN_SECRET_KEY", "sk1_foobarbaz")
	os.Setenv("PORKBUN_BASE_URL", testUrl)

	tests := []struct {
		name         string
		testResource string
		record       porkbun.Record
	}{
		{
			name: "withSubdomain",
			testResource: `
          resource "porkbun_dns_record" "test" {
            name =   "test"
            domain = "foobar.dev"
            content = "0.0.0.1"
            type = "A"
          }
				`,
			record: porkbun.Record{
				Name:    "test",
				ID:      "987",
				Type:    "A",
				Content: "0.0.0.1",
				TTL:     "600",
				Prio:    "",
				Notes:   "",
			},
		},
		{
			name: "withoutSubdomain",
			testResource: `
          resource "porkbun_dns_record" "test" {
            name = ""
            domain = "foobar.dev"
            content = "0.0.0.1"
            type = "A"
          }
				`,
			record: porkbun.Record{
				Name:    "foobar.dev",
				ID:      "987",
				Type:    "A",
				Content: "0.0.0.1",
				TTL:     "600",
				Prio:    "",
				Notes:   "",
			},
		},
	}

	for _, test := range tests {
		r := require.New(t)
		expectRequest(func(w http.ResponseWriter, req *http.Request) {
			r.Equal(http.MethodPost, req.Method)
			r.Equal("/dns/retrieve/foobar.dev", req.URL.Path)
			r.NoError(json.NewEncoder(w).Encode(&retrieveResponse{
				Status:  "SUCCESS",
				Records: []porkbun.Record{test.record},
			}))
		})
		expectRequest(func(w http.ResponseWriter, req *http.Request) {
			r.Equal(http.MethodPost, req.Method)
			r.Equal("/dns/create/foobar.dev", req.URL.Path)
			r.NoError(json.NewEncoder(w).Encode(&createResponse{
				Status: "SUCCESS",
				ID:     987,
			}))
		})

		for range []int{1, 2, 3} {
			expectRequest(func(w http.ResponseWriter, req *http.Request) {
				r.Equal(http.MethodPost, req.Method)
				r.Equal("/dns/retrieve/foobar.dev", req.URL.Path)
				r.NoError(json.NewEncoder(w).Encode(&retrieveResponse{
					Status:  "SUCCESS",
					Records: []porkbun.Record{test.record},
				}))
			})

		}

		expectRequest(func(w http.ResponseWriter, req *http.Request) {
			r.Equal(http.MethodPost, req.Method)
			r.Equal("/dns/delete/foobar.dev/987", req.URL.Path)
			r.NoError(json.NewEncoder(w).Encode(&deleteResponse{
				Status: "SUCCESS",
			}))
		})

		expectRequest(func(w http.ResponseWriter, req *http.Request) {
			r.Equal(http.MethodPost, req.Method)
			r.Equal("/dns/retrieve/foobar.dev", req.URL.Path)
			r.NoError(json.NewEncoder(w).Encode(&retrieveResponse{
				Status:  "SUCCESS",
				Records: []porkbun.Record{test.record},
			}))
		})

		// Run the terraform twice to ensure its idempotent
		resource.UnitTest(t, resource.TestCase{
			IsUnitTest: true,
			Steps: []resource.TestStep{
				{
					ProtoV6ProviderFactories: protoV6ProviderFactories(testUrl),
					Config:                   test.testResource,
					Check: resource.ComposeTestCheckFunc(
						resource.TestCheckResourceAttr("porkbun_dns_record.test", "content", "0.0.0.1"),
					),
				},
				{
					ProtoV6ProviderFactories: protoV6ProviderFactories(testUrl),
					Config:                   test.testResource,
					Check: resource.ComposeTestCheckFunc(
						resource.TestCheckResourceAttr("porkbun_dns_record.test", "content", "0.0.0.1"),
					),
				},
			},
		})
	}

}

func MockPorkbun(t *testing.T) (string, func(http.HandlerFunc), func()) {
	var (
		receivedCalls   int
		expectedCalls   []http.HandlerFunc
		addExpectedCall = func(h http.HandlerFunc) {
			expectedCalls = append(expectedCalls, h)
		}
		r = require.New(t)
	)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		r.Less(
			receivedCalls,
			len(expectedCalls),
			"did not expect another request",
		)

		expectedCalls[receivedCalls](w, req)

		receivedCalls++
	}))

	return ts.URL, addExpectedCall, func() {
		ts.Close()
		r.Equal(
			len(expectedCalls),
			receivedCalls,
			"expected one more request",
		)
	}
}

type TestHttpMock struct {
	server *httptest.Server
}
