package resp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestPageEcho 校验分页响应的 page/pageSize 回显兜底：
// 不带分页参数（<=0）时归一为第 1 页 / 每页 20，避免历史技术债里的 page=0。
func TestPageEcho(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cases := []struct {
		name               string
		inPage, inSize     int
		wantPage, wantSize int
	}{
		{"裸查询未带分页", 0, 0, 1, 20},
		{"负值归一", -3, -1, 1, 20},
		{"正常值原样透传", 2, 50, 2, 50},
		{"仅缺 page", 0, 100, 1, 100},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			Page(c, []int{}, 0, tc.inPage, tc.inSize)
			if w.Code != http.StatusOK {
				t.Fatalf("HTTP 状态期望 200，实际 %d", w.Code)
			}
			var body struct {
				Code int      `json:"code"`
				Data PageData `json:"data"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatalf("响应解析失败: %v", err)
			}
			if body.Data.Page != tc.wantPage || body.Data.PageSize != tc.wantSize {
				t.Errorf("期望 page=%d size=%d，实际 page=%d size=%d",
					tc.wantPage, tc.wantSize, body.Data.Page, body.Data.PageSize)
			}
		})
	}
}
