package xrpath

func BuildResourceStates(world World, params Parameters, visibility VisibilitySet, tracking TrackingState, missing map[string]bool) []ResourceState {
	usedBy := make(map[string][]string)
	addUse := func(resourceID, eye string) {
		usedBy[resourceID] = appendUnique(usedBy[resourceID], eye)
	}

	for _, obj := range visibility.Left {
		if worldObj, ok := world.ObjectByID(obj.ID); ok {
			addUse(worldObj.MeshResource, "left")
			addUse(worldObj.TextureResource, "left")
		}
	}
	for _, obj := range visibility.Right {
		if worldObj, ok := world.ObjectByID(obj.ID); ok {
			addUse(worldObj.MeshResource, "right")
			addUse(worldObj.TextureResource, "right")
		}
	}
	if tracking.MatchedKeyframe != "" {
		addUse(tracking.MatchedKeyframe, "tracking")
	}
	if params.ReprojectionPolicy == "depth_motion_disocclusion" {
		addUse("warp-mesh-depth-motion", "reprojection")
	} else {
		addUse("warp-mesh-auto", "reprojection")
	}

	states := make([]ResourceState, 0, len(world.Resources))
	for _, spec := range world.Resources {
		users := append([]string{}, usedBy[spec.ID]...)
		if users == nil {
			users = []string{}
		}
		revision := 1
		if spec.ID == "warp-mesh-depth-motion" && params.Revision >= 2 {
			revision = 2
		}
		hash := resourceContentHash(spec, revision)
		state := ResourceState{
			ID: spec.ID, Kind: spec.Kind, Revision: revision, SHA256: hash,
			Status: "cached", Resident: true, Bytes: spec.Bytes,
			ReferenceCount: len(users), UsedBy: users,
		}

		if missing[spec.ID] {
			state.Status = "missing"
			state.Resident = false
			state.ReferenceCount = 0
			state.UsedBy = []string{}
		} else if len(users) > 0 {
			state.Status = "ready"
		} else if spec.Kind == "relocalization_keyframe" {
			if tracking.Status == "tracking" && tracking.MatchedKeyframe != "" {
				state.Status = "locked"
			} else {
				state.Status = "cached"
			}
		}
		states = append(states, state)
	}
	return states
}

func appendUnique(items []string, item string) []string {
	for _, existing := range items {
		if existing == item {
			return items
		}
	}
	return append(items, item)
}
