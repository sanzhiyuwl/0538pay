package service

import "testing"

// TestMergeSubChannelConfig 校验 B1-34 子通道 info 占位覆盖（对齐 epay Channel::getSub）。
func TestMergeSubChannelConfig(t *testing.T) {
	// 主通道 config：appid/mch_id 为占位 [xxx]，private_key 为固定值（不覆盖）。
	main := `{"appid":"[appid]","mch_id":"[mchid]","private_key":"SHARED_KEY","notify_url":"https://p.x/cb"}`
	// 子通道 info：提供占位实际值。
	sub := `{"appid":"2021000123","mchid":"1600001111"}`
	kv := mergeSubChannelConfig(main, sub)
	if kv == nil {
		t.Fatal("合并结果不应为 nil")
	}
	if kv["appid"] != "2021000123" {
		t.Errorf("appid 占位未替换：%q", kv["appid"])
	}
	if kv["mch_id"] != "1600001111" {
		t.Errorf("mch_id 占位未替换：%q", kv["mch_id"])
	}
	if kv["private_key"] != "SHARED_KEY" {
		t.Errorf("非占位键不应被改：%q", kv["private_key"])
	}
	if kv["notify_url"] != "https://p.x/cb" {
		t.Errorf("非占位键不应被改：%q", kv["notify_url"])
	}

	// 占位键在子通道 info 里缺失 → 保留原占位串（对齐 epay arr[key] 缺失即 null，此处保守保留）。
	kv2 := mergeSubChannelConfig(`{"appid":"[appid]"}`, `{"other":"x"}`)
	if kv2["appid"] != "[appid]" {
		t.Errorf("info 缺 key 时应保留占位：%q", kv2["appid"])
	}

	// 非法 JSON → nil（调用方退回主通道 config）。
	if mergeSubChannelConfig("not-json", sub) != nil || mergeSubChannelConfig(main, "not-json") != nil {
		t.Error("非法 JSON 应返回 nil")
	}
}
