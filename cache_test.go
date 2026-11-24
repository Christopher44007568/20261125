package eyebw

import "testing"

func TestCacheHitHasZeroTransferAndDoesNotDoubleCount(t *testing.T) {
	meter := NewMeter(0, PipelineSeparateEyes, NewCache(1024*MebiByte))

	first := meter.Read(StageHistory, EyeLeft, ClassHistory, "history:tile", 4096)
	second := meter.Read(StageHistory, EyeLeft, ClassHistory, "history:tile", 4096)
	thirdWrite := meter.Write(StageHistory, EyeLeft, ClassHistory, "history:tile", 4096)

	if first.CacheHit || first.TransferBytes != 4096 {
		t.Fatalf("first read should miss and transfer 4096 bytes: %+v", first)
	}
	if !second.CacheHit || second.TransferBytes != 0 {
		t.Fatalf("second read should hit and transfer zero bytes: %+v", second)
	}
	if thirdWrite.TransferBytes != 4096 {
		t.Fatalf("write-through must count its own transfer once: %+v", thirdWrite)
	}

	agg := AggregateAccesses(meter.Accesses)
	if agg.CacheHitBytes != 4096 {
		t.Fatalf("hit bytes = %d, want 4096", agg.CacheHitBytes)
	}
	if agg.TransferReadBytes != 4096 {
		t.Fatalf("transfer reads = %d, want only first miss", agg.TransferReadBytes)
	}
	if agg.TransferWriteBytes != 4096 {
		t.Fatalf("transfer writes = %d, want one write", agg.TransferWriteBytes)
	}
	if agg.TransferTotal() != 8192 {
		t.Fatalf("transfer total = %d, want 8192", agg.TransferTotal())
	}
}

func TestUncachedReadAlwaysTransfers(t *testing.T) {
	cache := NewCache(1024 * MebiByte)
	meter := NewMeter(0, PipelineSeparateEyes, cache)
	key := "front:left:0:0"
	meter.Write(StageReproject, EyeLeft, ClassColor, key, 128)
	scanout := meter.ReadUncached(StageScanout, EyeLeft, ClassColor, key, "display", 128)

	if scanout.CacheHit || scanout.TransferBytes != 128 {
		t.Fatalf("scanout must transfer even when cache resident: %+v", scanout)
	}
}

func TestLRUEvictionCausesANewMissWithoutPhantomHit(t *testing.T) {
	cache := NewCache(64)
	cache.Put("a", 32)
	cache.Put("b", 32)
	if _, ok := cache.Get("a"); !ok {
		t.Fatal("a should remain after two 32-byte entries fit in 64 bytes")
	}
	cache.Put("c", 32)
	if _, ok := cache.Get("b"); ok {
		t.Fatal("b should have been evicted after touching a and inserting c")
	}
	if _, ok := cache.Get("a"); !ok {
		t.Fatal("a was recently used and should remain")
	}
}
