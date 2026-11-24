// Package stereorollback is a standard-library-only simulation of stereo
// frame rendering with isolated per-eye failure and rollback.
//
// When exactly one eye fails (attachment creation, material parameters or
// reprojection timeout), the pipeline rolls back only that eye attempt,
// presents its declared placeholder and preserves the other eye's real frame.
// Shared resources live in a reference-counted ResourcePool: releasing one
// eye's leases never destroys a resource that still has other users, and a
// corrupt shared texture is handled by a separate quarantine/restore path.
//
// RunAll executes the three named scenarios (left-eye failure, right-eye
// failure and shared-texture corruption) and returns a JSON-friendly Report
// recording failure stages, retained resources, fallback frames and success
// frame counts. Placeholder surfaces are never counted as successes.
package stereorollback
