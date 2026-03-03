//go:build soak && !cloud

package v2

import (
	"context"
	stderrors "errors"
	"fmt"
	"testing"
)

func TestPerfShouldIgnoreContextError(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		opErr     error
		runCtxErr error
		opCtxErr  error
		want      bool
	}{
		{
			name:      "nil operation error",
			opErr:     nil,
			runCtxErr: context.Canceled,
			opCtxErr:  context.Canceled,
			want:      false,
		},
		{
			name:      "nil run context error",
			opErr:     context.Canceled,
			runCtxErr: nil,
			opCtxErr:  context.Canceled,
			want:      false,
		},
		{
			name:      "nil operation context error",
			opErr:     context.Canceled,
			runCtxErr: context.Canceled,
			opCtxErr:  nil,
			want:      false,
		},
		{
			name:      "direct canceled error",
			opErr:     context.Canceled,
			runCtxErr: context.Canceled,
			opCtxErr:  context.Canceled,
			want:      true,
		},
		{
			name:      "direct deadline exceeded error",
			opErr:     context.DeadlineExceeded,
			runCtxErr: context.DeadlineExceeded,
			opCtxErr:  context.DeadlineExceeded,
			want:      true,
		},
		{
			name:      "wrapped canceled error via errors is",
			opErr:     fmt.Errorf("request failed: %w", context.Canceled),
			runCtxErr: context.Canceled,
			opCtxErr:  context.Canceled,
			want:      true,
		},
		{
			name:      "wrapped deadline exceeded error via errors is",
			opErr:     fmt.Errorf("request failed: %w", context.DeadlineExceeded),
			runCtxErr: context.DeadlineExceeded,
			opCtxErr:  context.DeadlineExceeded,
			want:      true,
		},
		{
			name:      "mismatched context kind",
			opErr:     context.Canceled,
			runCtxErr: context.Canceled,
			opCtxErr:  context.DeadlineExceeded,
			want:      false,
		},
		{
			name:      "exact canceled message fallback",
			opErr:     stderrors.New("context canceled"),
			runCtxErr: context.Canceled,
			opCtxErr:  context.Canceled,
			want:      true,
		},
		{
			name:      "suffix deadline message fallback",
			opErr:     stderrors.New("transport timeout: context deadline exceeded"),
			runCtxErr: context.DeadlineExceeded,
			opCtxErr:  context.DeadlineExceeded,
			want:      true,
		},
		{
			name:      "contains context text but not suffix",
			opErr:     stderrors.New("context canceled while writing wal record"),
			runCtxErr: context.Canceled,
			opCtxErr:  context.Canceled,
			want:      false,
		},
		{
			name:      "unrelated error",
			opErr:     stderrors.New("integrity check failed"),
			runCtxErr: context.Canceled,
			opCtxErr:  context.Canceled,
			want:      false,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := perfShouldIgnoreContextError(tc.opErr, tc.runCtxErr, tc.opCtxErr)
			if got != tc.want {
				t.Fatalf("perfShouldIgnoreContextError()=%t want %t", got, tc.want)
			}
		})
	}
}
