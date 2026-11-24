package xrpath

func scriptedEventsForFrame(frame int) []Event {
	t := float64(frame) / FrameRateHz
	switch frame {
	case PauseAtFrame:
		return []Event{{Frame: frame, AtSeconds: t, Type: "pause", Detail: "pause requested; pose held"}}
	case ReloadAtFrame:
		return []Event{{
			Frame: frame, AtSeconds: t, Type: "parameter_reload_requested",
			Detail: "tuning-v2 staged while paused", Revision: 2,
		}}
	case ResumeAtFrame:
		return []Event{
			{Frame: frame, AtSeconds: t, Type: "resume", Detail: "motion clock continues without a frame jump"},
			{Frame: frame, AtSeconds: t, Type: "parameter_reloaded", Detail: "tuning-v2 active", Revision: 2},
		}
	default:
		return nil
	}
}

func buildManifest(records []FrameRecord, events []Event) Manifest {
	phases := []PhaseSummary{}
	for _, record := range records {
		if len(phases) == 0 || phases[len(phases)-1].Name != record.Phase {
			phases = append(phases, PhaseSummary{
				Name: record.Phase, FirstFrame: record.Frame, LastFrame: record.Frame,
			})
		} else {
			phases[len(phases)-1].LastFrame = record.Frame
		}
	}

	return Manifest{
		SchemaVersion:   SchemaVersion,
		FrameCount:      len(records),
		FrameRateHz:     FrameRateHz,
		FixedPathSHA256: FixedPathHash(),
		Parameters:      ReloadedParameters(),
		Phases:          phases,
		Events:          events,
	}
}
