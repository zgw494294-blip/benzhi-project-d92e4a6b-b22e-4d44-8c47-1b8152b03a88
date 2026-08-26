package main

import "testing"

func TestAddressConfiguration(t *testing.T) {
	t.Setenv("PORT", "19222")
	value, err := parseConfig(nil)
	if err != nil || value.Address != "127.0.0.1:19222" {
		t.Fatalf("PORT 解析失败: %+v %v", value, err)
	}
	value, err = parseConfig([]string{"-addr=127.0.0.1:19333"})
	if err != nil || value.Address != "127.0.0.1:19333" {
		t.Fatalf("显式 addr 未优先: %+v %v", value, err)
	}
	if _, err := parseConfig([]string{"-addr=0.0.0.0:19081"}); err == nil {
		t.Fatal("不安全监听地址未拒绝")
	}
	if _, err := parseConfig([]string{"-addr=127.0.0.1:8080"}); err == nil {
		t.Fatal("常见低位端口未拒绝")
	}
}
