package expletives

import "sort"

func layoutIDForControl(state *controlState) LayoutID {
	if state.layout == nil {
		return ""
	}
	return state.layout.id
}

func layoutArrangementIndex(state *controlState) int {
	if state.layout == nil {
		return -1
	}
	for index, item := range state.layout.items {
		if item.kind == layoutPanelItem && item.panel == state {
			return index
		}
	}
	return -1
}

func layoutStackIndex(state *controlState) int {
	if state.layout == nil {
		return -1
	}
	for index, item := range state.layout.stack {
		if item.kind == layoutPanelItem && item.panel == state {
			return index
		}
	}
	return -1
}

func (a *App) layoutSnapshotsLocked() []LayoutSnapshot {
	ids := make([]string, 0, len(a.layoutsByID))
	for id := range a.layoutsByID {
		ids = append(ids, string(id))
	}
	sort.Strings(ids)
	snapshots := make([]LayoutSnapshot, 0, len(ids))
	for _, encoded := range ids {
		state := a.layoutsByID[LayoutID(encoded)]
		if state == nil || state.destroyed {
			continue
		}
		parent := LayoutID("")
		if state.parent != nil {
			parent = state.parent.id
		}
		items := make([]LayoutItemSnapshot, 0, len(state.items))
		for arrangementIndex, item := range state.items {
			snapshot := LayoutItemSnapshot{
				Bounds:      item.bounds,
				Minimum:     item.minimum,
				LayoutIndex: arrangementIndex,
				StackIndex:  layoutItemStackIndex(state, item),
			}
			if item.kind == layoutPanelItem {
				snapshot.Kind = "panel"
				snapshot.Panel = item.panel.id
			} else {
				snapshot.Kind = "layout"
				snapshot.Layout = item.layout.id
			}
			items = append(items, snapshot)
		}
		border := state.border.details().Border
		border.ResolvedStyle = resolveBorderStyle(
			a.styles[border.Style],
			state.border,
		)
		snapshots = append(snapshots, LayoutSnapshot{
			ID:          state.id,
			Key:         state.automationKey,
			Kind:        state.kind,
			Owner:       state.owner.id,
			Parent:      parent,
			Bounds:      state.bounds,
			OwnerBounds: state.ownerBounds,
			Minimum:     state.minimum,
			Border:      border,
			LayoutIndex: layoutArrangementIndexForLayout(state),
			StackIndex:  layoutStackIndexForLayout(state),
			Items:       items,
		})
	}
	return snapshots
}

func layoutArrangementIndexForLayout(state *layoutState) int {
	if state.parent == nil {
		return state.rootIndex
	}
	for index, item := range state.parent.items {
		if item.kind == layoutLayoutItem && item.layout == state {
			return index
		}
	}
	return -1
}

func layoutStackIndexForLayout(state *layoutState) int {
	if state.parent == nil {
		for index, layout := range state.owner.layoutRoots {
			if layout == state {
				return index
			}
		}
		return -1
	}
	for index, item := range state.parent.stack {
		if item.kind == layoutLayoutItem && item.layout == state {
			return index
		}
	}
	return -1
}

func layoutItemStackIndex(state *layoutState, target *layoutItem) int {
	for index, item := range state.stack {
		if item == target {
			return index
		}
	}
	return -1
}
