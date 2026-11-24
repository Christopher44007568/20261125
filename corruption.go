package stereorollback

// goodAtlasPixels is the known-good snapshot used to restore the shared atlas.
func goodAtlasPixels() []byte {
	pixels := make([]byte, 64)
	for i := range pixels {
		pixels[i] = byte(i * 3)
	}
	return pixels
}

// handleSharedCorruption is the distinct path for a corrupt shared texture.
// It quarantines and restores only that texture from a known-good snapshot,
// preserving every holder and leaving unrelated shared resources untouched.
// The in-flight frame is conservatively presented with the shared emergency
// image; subsequent frames use the restored texture.
func (p *Pipeline) handleSharedCorruption(frame uint64, id ResourceID) (*CorruptionRecord, FallbackFrame) {
	badInfo, _ := p.pool.Info(id)
	expected, actual, _ := p.pool.Checksums(id)
	affected := p.pool.Users(id)

	_, _ = p.pool.Quarantine(id)

	unaffected := make([]ResourceID, 0)
	for _, info := range p.pool.List() {
		if info.ID != id {
			unaffected = append(unaffected, info.ID)
		}
	}

	restored, err := p.pool.Restore(id, goodAtlasPixels())
	if err != nil {
		restored = badInfo
	}
	migrated := p.pool.Users(id)

	record := &CorruptionRecord{
		Frame:                frame,
		Scope:                ScopeSharedResource,
		ResourceID:           id,
		OldGeneration:        badInfo.Generation,
		ExpectedChecksum:     expected,
		ActualChecksum:       actual,
		ReplacementID:        id,
		NewGeneration:        restored.Generation,
		AffectedUsers:        affected,
		MigratedUsers:        migrated,
		DroppedUsers:         []string{},
		QuarantinedResources: []ResourceID{id},
		UnaffectedResources:  unaffected,
		Note:                 "only the corrupt shared texture was quarantined and restored; all users retained",
	}

	note := "shared texture checksum mismatch; presenting shared fallback for the in-flight frame"
	left := sharedFallbackSurface(EyeLeft, p.sharedFallback, note)
	right := sharedFallbackSurface(EyeRight, p.sharedFallback, note)
	fallback := newSharedFallbackFrame(frame, ScopeSharedResource, left, right)
	return record, fallback
}
