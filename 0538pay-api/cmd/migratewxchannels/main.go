// migratewxchannels 把存量「微信形态包」通道（wxnative/wxjsapi/wxh5/wxapp 及 wxv2* 系列）
// 就地迁移为「聚合门面」通道（wxpay / wxpayv2），并按原形态回填 apptype。
//
// 背景（批次二，配合微信通道后端合并）：批次一新增 wxpay/wxpayv2 门面渠道，一个通道承载全形态、
// 下单按买家场景分派。存量通道每条只绑一个形态（plugin=wxnative 等），需迁到门面 key 并把
// 该形态记进 apptype，下单才走门面分派。
//
// 迁移策略（安全第一，见三问·扩散）：
//   - 【就地转换，不合并】每条形态通道 1:1 改 plugin→门面 key、apptype←该形态编码。通道 id 不变，
//     故 pay_subchannel.channel / 订单 order.channel 等外键引用全部保持有效，无需连带改。
//   - 【不自动合并多条】不同通道可能配不同密钥（不同服务商/商户号），自动并会串号。若需把同一
//     服务商的多形态通道并成一条多形态通道，由管理员在后台手工调整 apptype，本脚本不做。
//   - 【幂等】已是门面 key 的通道跳过；apptype 已含该形态则不重复加。可反复执行。
//
// 用法（务必先备份 pay_channel）：
//
//	go run ./cmd/migratewxchannels -config ./configs            # 预演（dry-run，只打印不改库）
//	go run ./cmd/migratewxchannels -config ./configs -apply     # 真正执行
package main

import (
	"flag"
	"log"
	"strings"

	"github.com/epvia/api/internal/config"
	"github.com/epvia/api/internal/model"
)

// formPluginToFacade 形态包 plugin → 门面 plugin + 该形态在 epay 语义下的 apptype 编码。
// apptype 编码对齐 epay $info['select']：1=Native 2=JSAPI 3=H5 5=APP 6=付款码。
var formPluginToFacade = map[string]struct {
	Facade  string
	AppType string
}{
	// APIv3 → wxpay
	"wxnative": {"wxpay", "1"},
	"wxjsapi":  {"wxpay", "2"},
	"wxh5":     {"wxpay", "3"},
	"wxapp":    {"wxpay", "5"},
	// APIv2 → wxpayv2
	"wxv2native": {"wxpayv2", "1"},
	"wxv2jsapi":  {"wxpayv2", "2"},
	"wxv2h5":     {"wxpayv2", "3"},
	"wxv2app":    {"wxpayv2", "5"},
	"wxv2micro":  {"wxpayv2", "6"},
}

// mergeAppType 把 code 并进逗号分隔的 apptype 串（去重、保序）。
func mergeAppType(cur, code string) string {
	set := map[string]bool{}
	out := []string{}
	for _, p := range strings.Split(cur, ",") {
		if v := strings.TrimSpace(p); v != "" && !set[v] {
			set[v] = true
			out = append(out, v)
		}
	}
	if !set[code] {
		out = append(out, code)
	}
	return strings.Join(out, ",")
}

func main() {
	configPath := flag.String("config", "./configs", "配置目录路径")
	apply := flag.Bool("apply", false, "真正执行迁移（缺省为 dry-run 预演，只打印不改库）")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}
	db, err := model.NewDB(cfg.Database)
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}

	var list []model.Channel
	if err := db.Where("plugin IN ?", keysOf(formPluginToFacade)).Order("id ASC").Find(&list).Error; err != nil {
		log.Fatalf("查询存量微信形态通道失败: %v", err)
	}
	if len(list) == 0 {
		log.Println("无存量微信形态通道（plugin=wxnative/wxjsapi/… 均无），无需迁移。")
		return
	}

	mode := "【预演 dry-run】"
	if *apply {
		mode = "【执行 apply】"
	}
	log.Printf("%s 命中 %d 条形态通道，逐条就地转门面 key + 回填 apptype：", mode, len(list))

	migrated := 0
	for i := range list {
		c := &list[i]
		m, ok := formPluginToFacade[c.Plugin]
		if !ok {
			continue // 理论不达（查询已过滤），防御
		}
		newApp := mergeAppType(c.AppType, m.AppType)
		log.Printf("  id=%d name=%q  plugin %s→%s  apptype %q→%q",
			c.ID, c.Name, c.Plugin, m.Facade, c.AppType, newApp)
		if !*apply {
			continue
		}
		if err := db.Model(&model.Channel{}).Where("id = ?", c.ID).
			Updates(map[string]interface{}{"plugin": m.Facade, "apptype": newApp}).Error; err != nil {
			log.Printf("  ✗ id=%d 迁移失败: %v", c.ID, err)
			continue
		}
		migrated++
	}

	if *apply {
		log.Printf("迁移完成：成功 %d / 命中 %d。请抓一次真实下单报文验证形态分派与子商户号正确。", migrated, len(list))
	} else {
		log.Printf("预演结束（未改库）。确认无误后加 -apply 执行；执行前务必先备份 pay_channel。")
	}
}

func keysOf(m map[string]struct {
	Facade  string
	AppType string
}) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	return ks
}
