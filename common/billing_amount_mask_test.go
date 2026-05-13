package common

import (
	"strings"
	"testing"
)

func TestMaskBillingAmountsForClient(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "chinese pre consume quota with currency",
			in:   "status_code=403, 预扣费额度失败, 用户剩余额度: ¥0.056700, 需要预扣费额度: ¥0.069900 (request id req_123)",
			want: "status_code=403, 预扣费额度失败, 用户剩余额度: ¥***, 需要预扣费额度: ¥*** (request id req_123)",
		},
		{
			name: "chinese pre consume quota with fullwidth dollar",
			in:   "status_code=403, 预扣费额度失败, 用户剩余额度: ＄0.053570, 需要预扣费额度: ＄0.060000",
			want: "status_code=403, 预扣费额度失败, 用户剩余额度: ＄***, 需要预扣费额度: ＄***",
		},
		{
			name: "english quota labels without currency",
			in:   "token quota is not enough, token remain quota: 120, need quota: 300",
			want: "token quota is not enough, token remain quota: ***, need quota: ***",
		},
		{
			name: "billing label with custom unit prefix",
			in:   "upstream balance: credits 12.50, request id req_123",
			want: "upstream balance: credits ***, request id req_123",
		},
		{
			name: "billing label with bracket prefix",
			in:   "upstream balance: (estimated) 12.50, request id req_123",
			want: "upstream balance: (estimated) ***, request id req_123",
		},
		{
			name: "billing label with numeric prefix",
			in:   "upstream balance: tier 2 credits 12.50, request id req_123",
			want: "upstream balance: tier 2 credits ***, request id req_123",
		},
		{
			name: "billing amount before numeric request id",
			in:   "upstream balance: credits 12.50 request id 123",
			want: "upstream balance: credits *** request id 123",
		},
		{
			name: "subscription need equals",
			in:   "subscription quota insufficient, need=69900",
			want: "subscription quota insufficient, need=***",
		},
		{
			name: "unrelated status code and request id numbers stay visible",
			in:   "status_code=403, upstream failed (request id req_123456)",
			want: "status_code=403, upstream failed (request id req_123456)",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := MaskBillingAmountsForClient(tt.in)
			if got != tt.want {
				t.Fatalf("MaskBillingAmountsForClient() = %q, want %q", got, tt.want)
			}
			if strings.Contains(got, "0.056700") || strings.Contains(got, "0.069900") ||
				strings.Contains(got, "0.053570") || strings.Contains(got, "0.060000") ||
				strings.Contains(got, "12.50") {
				t.Fatalf("masked message still contains original amount: %q", got)
			}
		})
	}
}
