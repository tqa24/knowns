package mcp

import (
	"context"
	"time"

	"github.com/howznguyen/knowns/internal/lsp"
	"github.com/howznguyen/knowns/internal/storage"
	"github.com/mark3labs/mcp-go/server"
)

func newLifecycleHooks(auditStore *storage.AuditStore, getRoot func() string, getLSPManager func() *lsp.Manager) *server.Hooks {
	hooks := newAuditHooks(auditStore, getRoot)
	hooks.AddOnRegisterSession(func(ctx context.Context, session server.ClientSession) {
		mgr := getLSPManager()
		if mgr == nil {
			return
		}
		startCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		if err := mgr.ClientConnected(startCtx); err != nil {
			mcpLog.Printf("lsp start skipped: %v", err)
		}
	})
	hooks.AddOnUnregisterSession(func(ctx context.Context, session server.ClientSession) {
		mgr := getLSPManager()
		if mgr == nil {
			return
		}
		stopCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		if err := mgr.ClientDisconnected(stopCtx); err != nil {
			mcpLog.Printf("lsp stop skipped: %v", err)
		}
	})
	return hooks
}
