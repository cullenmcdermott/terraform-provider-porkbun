package provider

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func Test_CreateRecordWithSubdomainSuccess(t *testing.T) {
	lastOctet := randomOctet()
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testRecordConfigWithSubdomain(lastOctet),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("porkbun_dns_record.test", "content", fmt.Sprintf("0.0.0.%v", lastOctet)),
					resource.TestCheckResourceAttr("porkbun_dns_record.test", "name", fmt.Sprintf("%v-foo", lastOctet)),
				),
			},
		},
	})
}

func Test_CreateRecordWithoutSubdomainSuccess(t *testing.T) {
	lastOctet := randomOctet()
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testRecordConfigNoSubdomain(lastOctet),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("porkbun_dns_record.test", "content", fmt.Sprintf("0.0.0.%v", lastOctet)),
					resource.TestCheckResourceAttr("porkbun_dns_record.test", "name", ""),
				),
			},
		},
	})
}

//func Test_CreateRecordWithRetriesSuccess(t *testing.T) {
//  lastOctet := randomOctet()
//	resource.Test(t, resource.TestCase{
//		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
//		Steps: []resource.TestStep{
//			{
//				Config: testRecordWithRetries(lastOctet),
//				Check: resource.ComposeAggregateTestCheckFunc(
//					resource.TestCheckResourceAttr("porkbun_dns_record.test[0]", "content", fmt.Sprintf("0.0.0.%v", lastOctet)),
//					resource.TestCheckResourceAttr("porkbun_dns_record.test[0]", "name", fmt.Sprintf("%v-foo-0", lastOctet)),
//				),
//			},
//		},
//	})
//}

func testRecordConfigNoSubdomain(randomIp int) string {
	return fmt.Sprintf(`
resource "porkbun_dns_record" "test" {
  domain = "providertest.top"
  content = "0.0.0.%v"
  type = "A"
}
`, randomIp)
}

func testRecordConfigWithSubdomain(randomIp int) string {
	return fmt.Sprintf(`
resource "porkbun_dns_record" "test" {
  name = "%v-foo"
  domain = "providertest.top"
  content = "0.0.0.%v"
  type = "A"
}
`, randomIp, randomIp)
}

func testRecordWithRetries(randomIp int) string {
	return fmt.Sprintf(`
resource "porkbun_dns_record" "test" {
  count = 1
  name = "%v-foo-${count.index}"
  domain = "providertest.top"
  content = "0.0.0.%v"
  type = "A"
}
`, randomIp, randomIp)
}

func randomOctet() int {
	return rand.Intn(255-0) + 0
}
