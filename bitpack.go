package main

// BitWriter packs n-bit unsigned integers tightly into a byte slice.
type BitWriter struct {
	buf   []byte
	acc   uint64
	nbits uint
}

func (w *BitWriter) WriteBits(v uint32, n uint) {
	w.acc |= uint64(v&((1<<n)-1)) << w.nbits
	w.nbits += n
	for w.nbits >= 8 {
		w.buf = append(w.buf, byte(w.acc))
		w.acc >>= 8
		w.nbits -= 8
	}
}

func (w *BitWriter) Bytes() []byte {
	if w.nbits > 0 {
		return append(w.buf, byte(w.acc))
	}
	return w.buf
}

// BitReader is the inverse of BitWriter.
type BitReader struct {
	buf   []byte
	pos   int // byte position
	acc   uint64
	nbits uint
}

func NewBitReader(buf []byte) *BitReader { return &BitReader{buf: buf} }

func (r *BitReader) ReadBits(n uint) uint32 {
	for r.nbits < n {
		r.acc |= uint64(r.buf[r.pos]) << r.nbits
		r.nbits += 8
		r.pos++
	}
	v := uint32(r.acc & ((1 << n) - 1))
	r.acc >>= n
	r.nbits -= n
	return v
}
