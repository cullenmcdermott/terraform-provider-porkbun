terraform {
  required_providers {
    porkbun = {
      source  = "cullenmcdermott/porkbun"
    }
  }
}

provider "porkbun" {}

resource "porkbun_dns_record" "test" {
  name = "foo"
  domain = "erinandcullen.com"
  content = "0.0.0.1"
  type = "A"
}
