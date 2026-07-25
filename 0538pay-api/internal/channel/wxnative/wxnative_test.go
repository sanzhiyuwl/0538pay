package wxnative

import (
	"testing"

	"github.com/epvia/api/internal/channel"
)

// TestKeyRegistered wxnative 通过 init() 自注册到 registry。
func TestKeyRegistered(t *testing.T) {
	if _, ok := channel.Get("wxnative"); !ok {
		t.Fatal("wxnative 未注册到 registry")
	}
}

// TestKeyAndProducts 声明的渠道 key 与支持的产品形态。
func TestKeyAndProducts(t *testing.T) {
	c := Channel{}
	if c.Key() != "wxnative" {
		t.Fatalf("Key 应为 wxnative, 实际 %s", c.Key())
	}
	products := c.Products()
	if len(products) != 1 || products[0].Code != "native" {
		t.Fatalf("应支持 native 一种产品, 实际 %v", products)
	}
}
