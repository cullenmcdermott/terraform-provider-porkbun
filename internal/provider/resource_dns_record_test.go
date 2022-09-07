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

// Status the API response status.
//type Status struct {
//	Status  string `json:"status"`
//	Message string `json:"message,omitempty"`
//}
//
//func (a Status) Error() string {
//	return fmt.Sprintf("%s: %s", a.Status, a.Message)
//}

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

	r := require.New(t)
	expectRequest(func(w http.ResponseWriter, req *http.Request) {
		r.Equal(http.MethodPost, req.Method)
		r.Equal("/dns/create/foobar.dev", req.URL.Path)
		r.NoError(json.NewEncoder(w).Encode(&createResponse{
			Status: "SUCCESS",
			ID:     987,
		}))
	})

	expectRequest(func(w http.ResponseWriter, req *http.Request) {
		r.Equal(http.MethodPost, req.Method)
		r.Equal("/dns/retrieve/foobar.dev", req.URL.Path)
		r.NoError(json.NewEncoder(w).Encode(&deleteResponse{
			Status: "SUCCESS",
		}))
	})

	expectRequest(func(w http.ResponseWriter, req *http.Request) {
		r.Equal(http.MethodPost, req.Method)
		r.Equal("/dns/delete/foobar.dev/987", req.URL.Path)
		r.NoError(json.NewEncoder(w).Encode(&retrieveResponse{
			Status: "SUCCESS",
			Records: []porkbun.Record{
				{
					Name:    "test",
					ID:      "987",
					Type:    "A",
					Content: "0.0.0.1",
					TTL:     "600",
					Prio:    "",
					Notes:   "",
				},
			},
		}))
	})
	resource.UnitTest(t, resource.TestCase{
		//ProtoV6ProviderFactories: protoV6ProviderFactories(testUrl),
		IsUnitTest: true,
		Steps: []resource.TestStep{
			{
				ProtoV6ProviderFactories: protoV6ProviderFactories(testUrl),
				Config: `
          resource "porkbun_dns_record" "test" {
            name = "test"
            domain = "foobar.dev"
            content = "0.0.0.1"
            type = "A"
          }
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("porkbun_dns_record.test", "content", "0.0.0.1"),
				),
			},
		},
	})
}

// removed *porkbun.Client from returns
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

	//parsedUrl, err := url.Parse(ts.URL)
	//if err != nil {
	//	panic(fmt.Sprintf("failed to parse test url: %s", err))
	//}

	//c := http.Client{}
	//porkbunTestClient := porkbun.Client{
	//	secretApiKey: "sk1_foobarbaz",
	//	apiKey:       "pk1_foobarbaz",
	//	BaseURL:      parsedUrl, // This will be returned by  the mock http server
	//	HTTPClient:   &http.Client{Timeout: 10 * time.Second},
	//}
	//url := url.Url{
	//	Host: ts.Url,
	//}
	// removed porkbunTestClient from return statement
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

func setUpMockHttpServer() *TestHttpMock {
	Server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			w.Header().Add("X-Single", "foobar")
			w.Header().Add("X-Double", "1")
			w.Header().Add("X-Double", "2")

			switch r.URL.Path {
			case "/200":
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("1.0.0"))
			case "/restricted":
				if r.Header.Get("Authorization") == "Zm9vOmJhcg==" {
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte("1.0.0"))
				} else {
					w.WriteHeader(http.StatusForbidden)
				}
			case "/utf-8/200":
				w.Header().Set("Content-Type", "text/plain; charset=UTF-8")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("1.0.0"))
			case "/utf-16/200":
				w.Header().Set("Content-Type", "application/json; charset=UTF-16")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("1.0.0"))
			case "/x509-ca-cert/200":
				w.Header().Set("Content-Type", "application/x-x509-ca-cert")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("pem"))
			case "/create":
				if r.Method == "POST" {
					w.WriteHeader(http.StatusCreated)
					_, _ = w.Write([]byte("created"))
				}
			case "/head":
				if r.Method == "HEAD" {
					w.WriteHeader(http.StatusOK)
				}
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}),
	)

	return &TestHttpMock{
		server: Server,
	}
}
