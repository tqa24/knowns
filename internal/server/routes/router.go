// Package routes wires all REST API handlers into the chi.Router provided by
// the server package.
package routes

import (
	"github.com/go-chi/chi/v5"
	"github.com/howznguyen/knowns/internal/storage"
)

// SetupRoutes registers all /api sub-routes onto r.
// The caller is responsible for mounting r at the /api prefix.
// manager may be nil when workspace switching is not needed (e.g. tests).
func SetupRoutes(r chi.Router, store *storage.Store, sse Broadcaster, projectRoot string, manager *storage.Manager) {
	// Tasks
	tr := &TaskRoutes{store: store, sse: sse}
	tr.Register(r)

	// Docs
	dr := &DocRoutes{store: store, sse: sse}
	dr.Register(r)

	// Config
	cr := &ConfigRoutes{store: store}
	cr.Register(r)

	// Time tracking
	timr := &TimeRoutes{store: store, sse: sse}
	timr.Register(r)

	// Search
	sr := &SearchRoutes{store: store}
	sr.Register(r)

	// Templates
	tmplr := &TemplateRoutes{store: store, sse: sse}
	tmplr.Register(r)

	// Validate
	vr := &ValidateRoutes{store: store}
	vr.Register(r)

	// Notify (MCP → Server notifications)
	nr := &NotifyRoutes{store: store, sse: sse}
	nr.Register(r)

	// Imports
	ir := &ImportRoutes{store: store, sse: sse}
	ir.Register(r)

	// Activities
	ar := &ActivityRoutes{store: store}
	ar.Register(r)

	// Chats
	chr := &ChatRoutes{
		store:       store,
		sse:         sse,
		projectRoot: projectRoot,
	}
	chr.Register(r)

	// Skills
	skr := NewSkillRoutes(projectRoot)
	skr.Register(r)

	// User-level preferences (cross-project)
	upr := &UserPrefsRoutes{store: storage.NewUserPrefsStore()}
	upr.Register(r)

	// Workspaces (multi-project management)
	if manager != nil {
		wsr := &WorkspaceRoutes{manager: manager, sse: sse}
		wsr.Register(r)
	}
}
