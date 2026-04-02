---
title: 'Learning: Knowledge Graph WebUI'
description: Patterns, decisions, and failures from implementing the Knowledge Graph WebUI with Cytoscape.js
createdAt: '2026-04-02T10:17:20.355Z'
updatedAt: '2026-04-02T10:17:20.355Z'
tags:
  - learning
  - graph
  - cytoscape
  - react
  - webui
  - full-stack
---

## Patterns

### Cytoscape.js Container Sizing
- **What:** The flex-1 wrapper div must be the direct Cytoscape container. Overlays (legend, detail panel, impact summary) must be sibling divs with `position: absolute`, not children inside the Cytoscape container.
- **Why:** Cytoscape takes ownership of all DOM children inside its container div and will wipe them on re-render. Any overlay rendered inside the container disappears.
- **When to use:** Any Cytoscape.js integration in a React app.

```tsx
// Correct structure
<div className="flex-1 min-h-0 relative">
  <div ref={containerRef} style={{ width: "100%", height: "100%" }} />  {/* Cytoscape renders here */}
  <div className="absolute top-4 right-4 ...">Detail panel</div>        {/* Overlay: sibling */}
  <div className="absolute bottom-4 left-4 ...">Legend</div>            {/* Overlay: sibling */}
</div>
```

### Always-Mounted Container for ResizeObserver
- **What:** Never early-return before mounting the Cytoscape container div. Render the container always; show loading spinners as absolute overlays on top.
- **Why:** If you `return <Spinner />` before the container div mounts, the ResizeObserver never fires and Cytoscape never gets dimensions. The graph renders 0×0.
- **Source:** @doc/specs/graph-intelligence-features

### Client-Side Graph Intelligence via Cytoscape API
- **What:** Search, impact analysis, and cluster detection are all implemented client-side using Cytoscape traversal APIs — no backend changes needed.
- **APIs used:**
  - `cy.$id(nodeId)` + `.closedNeighborhood()` for search neighbor highlighting
  - `cy.elements().bfs()` for 2-hop impact BFS traversal
  - `cy.elements().components()` for cluster detection
- **When to use:** Any graph intelligence feature that operates on already-fetched graph data.

### Graph Backend Shape
- **What:** `/api/graph` returns `{ nodes: [{id, label, type, data}], edges: [{id, source, target, type}] }`. Node IDs are prefixed: `task:`, `doc:`, `memory:`.
- **Why:** Prefixed IDs allow merging nodes from multiple entity types without ID collisions.
- **Mention edges:** Detected server-side by regex matching `@task-([a-z0-9]+)` and `@doc/([^\s\)]+)` in task/doc content. Deduplication via `deduplicateEdges()`.

### Cytoscape CSS Class Mode for Visual State
- **What:** Use `.addClass()` / `.removeClass()` with Cytoscape stylesheet classes for all visual state changes (hover, search highlight, impact, cluster).
- **Why:** Cleaner than per-node style overrides; supports easy `clearGraphMode()` by removing all state classes at once.
- **Key classes:** `.search-match`, `.search-neighbor`, `.impact-root`, `.impact-hop1`, `.impact-hop2`, `.cluster-colored`, `.dimmed`, `.highlighted`

---

## Decisions

### Cytoscape.js over react-force-graph-2d
- **Chose:** Cytoscape.js
- **Over:** react-force-graph-2d (canvas-based, D3 force layout)
- **Tag:** GOOD_CALL
- **Reason:** react-force-graph-2d had persistent container sizing issues (canvas never filled the parent) and limited support for CSS-class-based state management. Cytoscape.js offers imperative DOM control, rich traversal API (bfs, components), and reliable resize handling.

### `cose` layout over `fcose`
- **Chose:** Built-in `cose` layout
- **Over:** `cytoscape-fcose` plugin
- **Tag:** GOOD_CALL
- **Reason:** fcose requires an npm package (`cytoscape-fcose`) and extra registration. Built-in cose works without plugins and produces acceptable layouts. Avoid optional plugins unless the quality gap is significant.

### Isolated nodes hidden by default
- **Chose:** Toggle to show/hide isolated nodes (no edges)
- **Over:** Always showing them
- **Tag:** GOOD_CALL
- **Reason:** Large graphs become unreadable when flooded with isolated nodes. A toggle keeps the default view clean while letting users see everything when needed.

### Graph as read-only view
- **Chose:** Graph page is view-only (no editing from graph)
- **Over:** Inline editing from node click
- **Tag:** GOOD_CALL
- **Reason:** Editing via graph introduces complex UX. Clicking a node opens a detail panel with a link to navigate to the entity's full edit page.

---

## Failures

### Spread overwriting computed color
- **What:** `{ color: computedColor, ...n.data }` — data fields overwrote computed color.
- **Fix:** Move spread to beginning: `{ ...n.data, color: computedColor }`.
- **Lesson:** When building objects with both computed fields and spread data, always put the spread first so explicit fields win.

### Font size disappearing on zoom-in
- **What:** Label visibility check `fontSize >= 2.5` checked graph-coordinate size, not screen pixels. Zooming in made the threshold meaningless.
- **Fix:** `screenFontSize = fontSize * globalScale >= 8` — multiply by Cytoscape's `globalScale` to get screen pixels.
- **Lesson:** Any size threshold in Cytoscape canvas rendering must account for zoom level via `globalScale`.

### `fcose` layout error after package removal
- **What:** Removed `cytoscape-fcose` package but missed one `name: "fcose"` occurrence in the layout config, causing runtime error.
- **Fix:** Use `replace_all: true` when renaming config values across a file.
- **Lesson:** When removing a library, grep for all string references (not just imports) before finishing.

### Re-layout spreading nodes apart
- **What:** Calling `d3ReheatSimulation()` repeatedly added energy without resetting force strengths, spreading nodes further each time.
- **Fix:** Reset `charge.strength(-100)` and `link.distance(30)` before reheat, then `zoomToFit` after 1 second.
- **Lesson:** When re-simulating a graph layout, reset force parameters first to prevent energy accumulation.
