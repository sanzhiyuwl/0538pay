package service

import "testing"

// newCfgWith 构造一个仅带内存缓存的 ConfigService（不接 DB），用于纯逻辑测试。
func newCfgWith(cache map[string]string) *ConfigService {
	return &ConfigService{cache: cache}
}

// TestDisabledPlugins plugin_disabled 逗号列表 → 禁用集合（空/空白项忽略）。
func TestDisabledPlugins(t *testing.T) {
	cfg := newCfgWith(map[string]string{"plugin_disabled": "wxnative, alipayf2f ,,fuiou2"})
	dis := cfg.DisabledPlugins()
	for _, k := range []string{"wxnative", "alipayf2f", "fuiou2"} {
		if !dis[k] {
			t.Errorf("期望 %s 在禁用集合", k)
		}
	}
	if len(dis) != 3 {
		t.Errorf("禁用集合应为 3 项，实际 %d：%v", len(dis), dis)
	}
	if cfg.PluginEnabled("wxnative") {
		t.Error("wxnative 应为禁用")
	}
	if !cfg.PluginEnabled("vmq") {
		t.Error("未列出的 vmq 应为启用")
	}
}

// TestDisabledPluginsEmpty 空值=全部启用（默认语义）。
func TestDisabledPluginsEmpty(t *testing.T) {
	cfg := newCfgWith(map[string]string{"plugin_disabled": ""})
	if len(cfg.DisabledPlugins()) != 0 {
		t.Error("空 plugin_disabled 应为空集合")
	}
	if !cfg.PluginEnabled("anything") {
		t.Error("空禁用列表下任意插件应启用")
	}
}

// TestCheckPluginEnabled 收单选通道拦截：禁用插件报错，mock/空/启用放行。
func TestCheckPluginEnabled(t *testing.T) {
	cfg := newCfgWith(map[string]string{"plugin_disabled": "wxnative"})
	s := &PayService{cfg: cfg}

	if err := s.checkPluginEnabled("wxnative"); err == nil {
		t.Error("禁用插件 wxnative 应被拦截")
	}
	if err := s.checkPluginEnabled("alipayf2f"); err != nil {
		t.Errorf("启用插件 alipayf2f 不应拦截：%v", err)
	}
	if err := s.checkPluginEnabled("mock"); err != nil {
		t.Errorf("mock 兜底应放行：%v", err)
	}
	if err := s.checkPluginEnabled(""); err != nil {
		t.Errorf("空 plugin 应放行：%v", err)
	}

	// cfg 未注入时向后兼容放行。
	s2 := &PayService{}
	if err := s2.checkPluginEnabled("wxnative"); err != nil {
		t.Errorf("cfg=nil 应放行：%v", err)
	}
}
