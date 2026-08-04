package service

import (
	"testing"

	"github.com/epvia/api/internal/model"
)

// TestSnapHitFunction 覆盖硬锁核心判定：只在 state=controlled 且命中目标能力时才拦截。
func TestSnapHitFunction(t *testing.T) {
	cases := []struct {
		name  string
		snap  *model.ChannelControl
		funcs []string
		want  string // 期望命中的能力枚举（空=放行）
	}{
		{"nil快照放行", nil, []string{ctrlFnNoTransaction}, ""},
		{
			"controlled命中收单",
			&model.ChannelControl{State: model.ChannelControlControlled, LimitedFunctions: `["NO_TRANSACTION"]`},
			[]string{ctrlFnNoTransaction, ctrlFnNoTransactionAndRecharge},
			ctrlFnNoTransaction,
		},
		{
			"controlled命中关闭收单和充值",
			&model.ChannelControl{State: model.ChannelControlControlled, LimitedFunctions: `["NO_TRANSACTION_AND_RECHARGE"]`},
			[]string{ctrlFnNoTransaction, ctrlFnNoTransactionAndRecharge},
			ctrlFnNoTransactionAndRecharge,
		},
		{
			"controlled但能力不在目标集则放行",
			&model.ChannelControl{State: model.ChannelControlControlled, LimitedFunctions: `["NO_WITHDRAWAL"]`},
			[]string{ctrlFnNoTransaction},
			"",
		},
		{
			"delayed不锁(未到点)",
			&model.ChannelControl{State: model.ChannelControlDelayed, LimitedFunctions: `["NO_TRANSACTION"]`},
			[]string{ctrlFnNoTransaction},
			"",
		},
		{
			"normal不锁",
			&model.ChannelControl{State: model.ChannelControlNormal, LimitedFunctions: `[]`},
			[]string{ctrlFnNoTransaction},
			"",
		},
		{
			"退款能力命中",
			&model.ChannelControl{State: model.ChannelControlControlled, LimitedFunctions: `["NO_REFUND","NO_PROFIT_SHARING"]`},
			[]string{ctrlFnNoRefund},
			ctrlFnNoRefund,
		},
		{
			"脏JSON放行(不误拦)",
			&model.ChannelControl{State: model.ChannelControlControlled, LimitedFunctions: `{bad`},
			[]string{ctrlFnNoTransaction},
			"",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := snapHitFunction(c.snap, c.funcs...); got != c.want {
				t.Fatalf("snapHitFunction = %q, want %q", got, c.want)
			}
		})
	}
}

// TestGuardSubMchNilSafe 未注入 controls / 空子商户号时放行（非服务商单不受影响）。
func TestGuardSubMchNilSafe(t *testing.T) {
	var g *ChannelControlGuard // nil receiver
	if err := g.GuardSubMch("123", ctrlFnNoTransaction); err != nil {
		t.Fatalf("nil guard 应放行，得 %v", err)
	}
	g2 := &ChannelControlGuard{} // controls 为 nil
	if err := g2.GuardSubMch("123", ctrlFnNoTransaction); err != nil {
		t.Fatalf("空 controls 应放行，得 %v", err)
	}
	if err := g2.GuardSubMch("", ctrlFnNoTransaction); err != nil {
		t.Fatalf("空子商户号应放行，得 %v", err)
	}
}
