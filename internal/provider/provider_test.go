package provider

import (
	"net/url"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/nrdcg/porkbun"
)

func newPorkbunProvider(testUrl string) provider.Provider {
	client := porkbun.New("sk1_foobarbaz", "pk1_foobarbaz")
	client.BaseURL, _ = url.Parse(testUrl)
	return &porkbunProvider{
		client:     client,
		configured: true,
		version:    "test",
	}
}

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"porkbun": providerserver.NewProtocol6WithError(New("test")()),
}

func protoV6ProviderFactories(url string) map[string]func() (tfprotov6.ProviderServer, error) {
	return map[string]func() (tfprotov6.ProviderServer, error){
		"porkbun": providerserver.NewProtocol6WithError(newPorkbunProvider(url)),
	}
}
