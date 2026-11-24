package stereorollback

// Stable owner names for shared-resource lease accounting.
const (
	OwnerReprojector = "stage:reprojector"
	OwnerRecorder    = "service:recorder"
	OwnerThumbnail   = "service:thumbnail"
	OwnerPresenter   = "service:presenter"
)

// OwnerEye returns the stable owner identity for an eye.
func OwnerEye(eye Eye) string { return "eye:" + string(eye) }

// Standard shared resource ids used by the reference scenarios.
const (
	ResSharedAtlas    ResourceID = "tex.shared_atlas"
	ResSharedDepth    ResourceID = "tex.shared_depth"
	ResSharedMaterial ResourceID = "ubo.shared_material"
	ResSharedHistory  ResourceID = "tex.shared_history"
	ResPlaceholderL   ResourceID = "img.placeholder.left"
	ResPlaceholderR   ResourceID = "img.placeholder.right"
	ResSharedFallback ResourceID = "img.shared_fallback"
)

// Script deterministically injects faults into numbered frames.
type Script struct {
	// EyeFaults[frame][eye] is the fault to raise for that eye attempt.
	EyeFaults map[uint64]map[Eye]FaultType
	// ElapsedMS overrides simulated reprojection elapsed time.
	ElapsedMS map[uint64]map[Eye]int
	// CorruptSharedAt[frame] corrupts the given shared texture before validation.
	CorruptSharedAt map[uint64]ResourceID
}

// FaultFor returns the injected fault for frame/eye, or FaultNone.
func (s Script) FaultFor(frame uint64, eye Eye) FaultType {
	if byEye, ok := s.EyeFaults[frame]; ok {
		return byEye[eye]
	}
	return FaultNone
}

// ElapsedFor returns the reprojection elapsed time for frame/eye.
func (s Script) ElapsedFor(frame uint64, eye Eye, fallback int) int {
	if byEye, ok := s.ElapsedMS[frame]; ok {
		if ms, ok := byEye[eye]; ok {
			return ms
		}
	}
	return fallback
}

// CorruptAt returns the shared resource to corrupt at frame, if scheduled.
func (s Script) CorruptAt(frame uint64) (ResourceID, bool) {
	id, ok := s.CorruptSharedAt[frame]
	return id, ok
}

// Config configures a Pipeline.
type Config struct {
	Width             int
	Height            int
	ReprojectBudgetMS int
	Pool              *ResourcePool
	Script            Script
}

// Pipeline is the simulated stereo renderer with transactional per-eye
// rollback and a ref-counted shared ResourcePool.
type Pipeline struct {
	width  int
	height int
	budget int
	pool   *ResourcePool
	script Script

	placeholders   map[Eye]*PixelImage
	sharedFallback *PixelImage
	materials      map[Eye]MaterialParams
	persistent     []Lease
}

// NewPipeline constructs a pipeline and its placeholder image bank.
func NewPipeline(cfg Config) *Pipeline {
	if cfg.Width <= 0 {
		cfg.Width = 16
	}
	if cfg.Height <= 0 {
		cfg.Height = 16
	}
	if cfg.ReprojectBudgetMS <= 0 {
		cfg.ReprojectBudgetMS = 16
	}
	pool := cfg.Pool
	if pool == nil {
		pool = NewResourcePool()
	}
	leftPH := placeholderImage(EyeLeft)
	rightPH := placeholderImage(EyeRight)
	return &Pipeline{
		width:          cfg.Width,
		height:         cfg.Height,
		budget:         cfg.ReprojectBudgetMS,
		pool:           pool,
		script:         cfg.Script,
		placeholders:   map[Eye]*PixelImage{EyeLeft: leftPH, EyeRight: rightPH},
		sharedFallback: sharedFallbackImage(),
		materials: map[Eye]MaterialParams{
			EyeLeft:  DefaultMaterialParams(ResSharedAtlas),
			EyeRight: DefaultMaterialParams(ResSharedAtlas),
		},
	}
}

// NewStandardPipeline builds a pipeline and bootstraps the standard shared
// resources and their long-lived users.
func NewStandardPipeline(script Script) (*Pipeline, error) {
	p := NewPipeline(Config{Script: script})
	if err := p.Bootstrap(); err != nil {
		return nil, err
	}
	return p, nil
}

// Pool exposes the underlying resource pool for assertions and inspection.
func (p *Pipeline) Pool() *ResourcePool { return p.pool }

// Placeholder returns the declared placeholder image for an eye.
func (p *Pipeline) Placeholder(eye Eye) *PixelImage { return p.placeholders[eye] }

// Bootstrap creates the standard shared resources and registers their
// long-lived users. These users are exactly what rollback must not clear.
func (p *Pipeline) Bootstrap() error {
	defs := []ResourceDesc{
		{ID: ResSharedAtlas, Kind: KindSharedTexture, Shared: true, Width: 8, Height: 8, Pixels: goodAtlasPixels()},
		{ID: ResSharedDepth, Kind: KindSharedDepth, Shared: true, Width: p.width, Height: p.height},
		{ID: ResSharedMaterial, Kind: KindSharedMaterial, Shared: true},
		{ID: ResSharedHistory, Kind: KindSharedHistory, Shared: true, Width: p.width, Height: p.height},
		{ID: ResPlaceholderL, Kind: KindFallbackImage, Shared: true, Width: 16, Height: 16, Pixels: p.placeholders[EyeLeft].RGBA},
		{ID: ResPlaceholderR, Kind: KindFallbackImage, Shared: true, Width: 16, Height: 16, Pixels: p.placeholders[EyeRight].RGBA},
		{ID: ResSharedFallback, Kind: KindFallbackImage, Shared: true, Width: 16, Height: 16, Pixels: p.sharedFallback.RGBA},
	}
	for _, desc := range defs {
		if _, err := p.pool.Create(desc); err != nil {
			return err
		}
	}

	holds := []struct {
		id    ResourceID
		owner string
	}{
		{ResSharedAtlas, OwnerEye(EyeLeft)},
		{ResSharedAtlas, OwnerEye(EyeRight)},
		{ResSharedAtlas, OwnerReprojector},
		{ResSharedAtlas, OwnerRecorder},
		{ResSharedDepth, OwnerEye(EyeLeft)},
		{ResSharedDepth, OwnerEye(EyeRight)},
		{ResSharedMaterial, OwnerEye(EyeLeft)},
		{ResSharedMaterial, OwnerEye(EyeRight)},
		{ResSharedHistory, OwnerEye(EyeLeft)},
		{ResSharedHistory, OwnerReprojector},
		{ResPlaceholderL, OwnerPresenter},
		{ResPlaceholderR, OwnerPresenter},
		{ResSharedFallback, OwnerPresenter},
	}
	for _, h := range holds {
		lease, err := p.pool.Acquire(h.id, h.owner)
		if err != nil {
			return err
		}
		p.persistent = append(p.persistent, lease)
	}
	return nil
}
