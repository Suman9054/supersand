package store

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// ---- helpers ----

func makeRecordBytes(typ RecordType, payload []byte, corrupt bool) []byte {
	var hdr [headerSize]byte
	crc := crc32.NewIEEE()
	crc.Write([]byte{byte(typ)})
	crc.Write(payload)
	sum := crc.Sum32()
	if corrupt {
		sum ^= 0xffffffff
	}
	binary.LittleEndian.PutUint32(hdr[0:4], sum)
	hdr[4] = byte(typ)
	binary.LittleEndian.PutUint16(hdr[5:7], 0) // tests write no key
	binary.LittleEndian.PutUint32(hdr[7:11], uint32(len(payload)))
	out := append(hdr[:], payload...)
	return out
}

// replayDir reads every segment in dir in order, assigning monotonic seq numbers.
func replayDir(t *testing.T, dir string) ([]Record, uint64) {
	t.Helper()
	segs, err := listSegments(dir)
	if err != nil {
		t.Fatalf("listSegments: %v", err)
	}
	var recs []Record
	var seq uint64
	for _, s := range segs {
		f, err := os.Open(s.path)
		if err != nil {
			t.Fatalf("open %s: %v", s.path, err)
		}
		br := bufio.NewReaderSize(f, 64*1024)
		for {
			rec, _, err := readRecord(br)
			if err == io.EOF || err == ErrCorrupt {
				break
			}
			if err != nil {
				f.Close()
				t.Fatalf("readRecord %s: %v", s.path, err)
			}
			seq++
			rec.Seq = seq
			recs = append(recs, rec)
		}
		f.Close()
	}
	return recs, seq
}

func readOne(t *testing.T, data []byte) (Record, error) {
	t.Helper()
	br := bufio.NewReaderSize(bytes.NewReader(data), 64*1024)
	rec, _, err := readRecord(br)
	return rec, err
}

// ---- Open / basic ----

func TestOpenCreatesDirAndSegment(t *testing.T) {
	dir := t.TempDir()
	w, err := openwal(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	segs, err := listSegments(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(segs) != 1 {
		t.Fatalf("expected 1 segment, got %d", len(segs))
	}
	if w.LastSeq() != 0 {
		t.Fatalf("expected lastSeq 0, got %d", w.LastSeq())
	}
}

func TestOpenCreatesNestedDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "a", "b", "c")
	w, err := openwal(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	if _, err := os.Stat(filepath.Join(dir, "seg-")); err != nil && !os.IsNotExist(err) {
		// just ensure dir exists; segment prefix file may not exist as literal
	}
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("nested dir not created: %v", err)
	}
}

// ---- Write / seq assignment ----

func TestWriteAssignsSeqAndPersists(t *testing.T) {
	dir := t.TempDir()
	w, err := openwal(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	s1, err := w.Write(TypeData, []byte("a"), [16]byte{})
	if err != nil {
		t.Fatal(err)
	}
	s2, err := w.Write(TypeCommit, []byte("bb"), [16]byte{})
	if err != nil {
		t.Fatal(err)
	}
	s3, err := w.Write(TypeCheckpoint, []byte("ccc"), [16]byte{})
	if err != nil {
		t.Fatal(err)
	}
	if s1 != 1 || s2 != 2 || s3 != 3 {
		t.Fatalf("unexpected seqs: %d %d %d", s1, s2, s3)
	}
	if w.LastSeq() != 3 {
		t.Fatalf("expected lastSeq 3, got %d", w.LastSeq())
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	recs, seq := replayDir(t, dir)
	if seq != 3 {
		t.Fatalf("expected 3 records replayed, got %d", seq)
	}
	if recs[0].Type != TypeData || string(recs[0].Payload) != "a" {
		t.Fatalf("record 0 mismatch: %+v", recs[0])
	}
	if recs[1].Type != TypeCommit || string(recs[1].Payload) != "bb" {
		t.Fatalf("record 1 mismatch: %+v", recs[1])
	}
	if recs[2].Type != TypeCheckpoint || string(recs[2].Payload) != "ccc" {
		t.Fatalf("record 2 mismatch: %+v", recs[2])
	}
}

func TestWriteEmptyPayload(t *testing.T) {
	dir := t.TempDir()
	w, err := openwal(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	s, err := w.Write(TypeData, nil, [16]byte{})
	if err != nil {
		t.Fatal(err)
	}
	if s != 1 {
		t.Fatalf("expected seq 1, got %d", s)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	recs, seq := replayDir(t, dir)
	if seq != 1 || len(recs[0].Payload) != 0 {
		t.Fatalf("empty payload not preserved: %+v seq=%d", recs, seq)
	}
}

func TestWriteAfterClose(t *testing.T) {
	dir := t.TempDir()
	w, err := openwal(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	_, err = w.Write(TypeData, []byte("x"), [16]byte{})
	if err != ErrClosed {
		t.Fatalf("expected ErrClosed, got %v", err)
	}
}

func TestWriteTooLarge(t *testing.T) {
	dir := t.TempDir()
	w, err := openwal(Options{Dir: dir, SegmentMaxBytes: 5})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	_, err = w.Write(TypeData, []byte("x"), [16]byte{}) // recLen = 9+1 = 10 > 5
	if err != ErrTooLarge {
		t.Fatalf("expected ErrTooLarge, got %v", err)
	}
}

func TestWriteZeroLengthOnTooSmallSegment(t *testing.T) {
	dir := t.TempDir()
	w, err := openwal(Options{Dir: dir, SegmentMaxBytes: 5})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	// even an empty record is 9 bytes > 5
	_, err = w.Write(TypeData, nil, [16]byte{})
	if err != ErrTooLarge {
		t.Fatalf("expected ErrTooLarge for empty record, got %v", err)
	}
}

// ---- Segment rolling ----

func TestSegmentRolling(t *testing.T) {
	dir := t.TempDir()
	// each 1-byte payload record is exactly 10 bytes, fits exactly one per segment
	w, err := openwal(Options{Dir: dir, SegmentMaxBytes: 30})
	if err != nil {
		t.Fatal(err)
	}
	const n = 4
	for i := 0; i < n; i++ {
		if _, err := w.Write(TypeData, []byte("x"), [16]byte{}); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	segs, err := listSegments(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(segs) != n {
		t.Fatalf("expected %d segments, got %d", n, len(segs))
	}
	recs, seq := replayDir(t, dir)
	if seq != uint64(n) {
		t.Fatalf("expected %d records replayed, got %d", n, seq)
	}
	for _, r := range recs {
		if string(r.Payload) != "x" {
			t.Fatalf("unexpected payload %q", r.Payload)
		}
	}
}

func TestRollThenContinueSeq(t *testing.T) {
	dir := t.TempDir()
	w, err := openwal(Options{Dir: dir, SegmentMaxBytes: 30})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		if _, err := w.Write(TypeData, []byte("x"), [16]byte{}); err != nil {
			t.Fatal(err)
		}
	}
	last := w.LastSeq()
	if last != 3 {
		t.Fatalf("expected lastSeq 3, got %d", last)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
}

// ---- SyncOnWrite ----

func TestSyncOnWriteFlushes(t *testing.T) {
	dir := t.TempDir()
	w, err := openwal(Options{Dir: dir, SyncOnWrite: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write(TypeData, []byte("hello"), [16]byte{}); err != nil {
		t.Fatal(err)
	}
	// without Close, data should already be on disk (flushed+fsynced)
	recs, seq := replayDir(t, dir)
	if seq != 1 || string(recs[0].Payload) != "hello" {
		t.Fatalf("SyncOnWrite did not persist: %+v seq=%d", recs, seq)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestSyncExplicit(t *testing.T) {
	dir := t.TempDir()
	w, err := openwal(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write(TypeData, []byte("y"), [16]byte{}); err != nil {
		t.Fatal(err)
	}
	// buffered, not yet flushed
	f, err := os.Open(w.active.Name())
	if err != nil {
		t.Fatal(err)
	}
	info, _ := f.Stat()
	f.Close()
	if info.Size() == 0 {
		// possible but let's just ensure Sync then works
	}
	if err := w.Sync(); err != nil {
		t.Fatal(err)
	}
	_, seq := replayDir(t, dir)
	if seq != 1 {
		t.Fatalf("Sync did not persist record, got seq %d", seq)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestSyncAfterClose(t *testing.T) {
	dir := t.TempDir()
	w, err := openwal(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if err := w.Sync(); err != ErrClosed {
		t.Fatalf("expected ErrClosed from Sync, got %v", err)
	}
}

// ---- Close ----

func TestCloseIdempotent(t *testing.T) {
	dir := t.TempDir()
	w, err := openwal(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("second Close should be nil, got %v", err)
	}
}

// ---- Recovery across reopen ----

func TestReopenRecoversLastSeq(t *testing.T) {
	dir := t.TempDir()
	w, err := openwal(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 5; i++ {
		if _, err := w.Write(TypeData, []byte("x"), [16]byte{}); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	w2, err := openwal(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	defer w2.Close()
	if w2.LastSeq() != 5 {
		t.Fatalf("expected lastSeq 5 after reopen, got %d", w2.LastSeq())
	}
	s, err := w2.Write(TypeData, []byte("x"), [16]byte{})
	if err != nil {
		t.Fatal(err)
	}
	if s != 6 {
		t.Fatalf("expected next seq 6, got %d", s)
	}
}

// ---- Torn tail / corruption recovery ----

func TestTornTailRecovery(t *testing.T) {
	dir := t.TempDir()
	w, err := openwal(Options{Dir: dir, SegmentMaxBytes: 30})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 5; i++ {
		if _, err := w.Write(TypeData, []byte("x"), [16]byte{}); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	// append a torn (partial) record to the last segment
	segs, _ := listSegments(dir)
	last := segs[len(segs)-1]
	f, err := os.OpenFile(last.path, os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write([]byte{0x01, 0x02, 0x03, 0x04, 0x05}); err != nil {
		t.Fatal(err)
	}
	f.Close()

	w2, err := openwal(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	defer w2.Close()
	if w2.LastSeq() != 5 {
		t.Fatalf("expected lastSeq 5 after torn tail, got %d", w2.LastSeq())
	}
	_, seq := replayDir(t, dir)
	if seq != 5 {
		t.Fatalf("expected 5 valid records, got %d", seq)
	}
}

func TestScanSegmentValidEnd(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "seg-00000000000000000001.db")
	data := makeRecordBytes(TypeData, []byte("abc"), false)
	data = append(data, []byte{0x01, 0x02, 0x03}...) // partial header tail
	if err := os.WriteFile(p, data, 0o644); err != nil {
		t.Fatal(err)
	}
	count, validEnd, _, err := scanSegment(p)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected count 1, got %d", count)
	}
	if validEnd != int64(headerSize+3) {
		t.Fatalf("expected validEnd %d, got %d", headerSize+3, validEnd)
	}
}

func TestScanSegmentStopsAtCorruption(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "seg-00000000000000000001.db")
	data := makeRecordBytes(TypeData, []byte("abc"), false)
	data = append(data, makeRecordBytes(TypeCommit, []byte("dead"), true)...) // corrupt 2nd record
	if err := os.WriteFile(p, data, 0o644); err != nil {
		t.Fatal(err)
	}
	count, validEnd, _, err := scanSegment(p)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected count 1 (corrupt tail excluded), got %d", count)
	}
	if validEnd != int64(headerSize+3) {
		t.Fatalf("expected validEnd %d, got %d", headerSize+3, validEnd)
	}
}

func TestScanSegmentEmpty(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "seg-00000000000000000001.db")
	if err := os.WriteFile(p, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	count, validEnd, _, err := scanSegment(p)
	if err != nil {
		t.Fatal(err)
	}
	if count != 0 || validEnd != 0 {
		t.Fatalf("expected empty scan, got count=%d validEnd=%d", count, validEnd)
	}
}

// ---- readRecord variants ----

func TestReadRecordCleanEOF(t *testing.T) {
	_, err := readOne(t, nil)
	if err != io.EOF {
		t.Fatalf("expected io.EOF, got %v", err)
	}
}

func TestReadRecordValid(t *testing.T) {
	data := makeRecordBytes(TypeData, []byte("payload"), false)
	rec, err := readOne(t, data)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if rec.Type != TypeData || string(rec.Payload) != "payload" {
		t.Fatalf("record mismatch: %+v", rec)
	}
}

func TestReadRecordCorruptCRC(t *testing.T) {
	data := makeRecordBytes(TypeData, []byte("payload"), true)
	_, err := readOne(t, data)
	if err != ErrCorrupt {
		t.Fatalf("expected ErrCorrupt, got %v", err)
	}
}

func TestReadRecordTornHeader(t *testing.T) {
	data := makeRecordBytes(TypeData, []byte("payload"), false)
	data = data[:4] // only partial header
	_, err := readOne(t, data)
	if err != io.EOF {
		t.Fatalf("expected io.EOF for torn header, got %v", err)
	}
}

func TestReadRecordTornPayload(t *testing.T) {
	// header claims 8-byte payload but only 3 provided
	var hdr [headerSize]byte
	crc := crc32.NewIEEE()
	crc.Write([]byte{byte(TypeData)})
	crc.Write([]byte("short"))
	binary.LittleEndian.PutUint32(hdr[0:4], crc.Sum32())
	hdr[4] = byte(TypeData)
	binary.LittleEndian.PutUint16(hdr[5:7], 0)
	binary.LittleEndian.PutUint32(hdr[7:11], 8)
	data := append(hdr[:], []byte("shor")[:3]...) // only 3 of 8 bytes
	_, err := readOne(t, data)
	if err != io.EOF {
		t.Fatalf("expected io.EOF for torn payload, got %v", err)
	}
}

// ---- Truncate ----

func TestTruncateNothing(t *testing.T) {
	dir := t.TempDir()
	w, err := openwal(Options{Dir: dir, SegmentMaxBytes: 30})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		if _, err := w.Write(TypeData, []byte("x"), [16]byte{}); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Truncate(0); err != nil {
		t.Fatal(err)
	}
	segs, _ := listSegments(dir)
	if len(segs) != 3 {
		t.Fatalf("expected 3 segments (nothing dropped), got %d", len(segs))
	}
	d, err := readDroppedSeq(dir)
	if err != nil {
		t.Fatal(err)
	}
	if d != 0 {
		t.Fatalf("expected droppedSeq 0, got %d", d)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestTruncateDropsOldSegments(t *testing.T) {
	dir := t.TempDir()
	w, err := openwal(Options{Dir: dir, SegmentMaxBytes: 30})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 5; i++ {
		if _, err := w.Write(TypeData, []byte("x"), [16]byte{}); err != nil {
			t.Fatal(err)
		}
	}
	// 5 segments: seqs 1..5. Truncate seq 3 drops seg1,2,3 (high-water <=3).
	if err := w.Truncate(3); err != nil {
		t.Fatal(err)
	}
	segs, _ := listSegments(dir)
	if len(segs) != 2 {
		t.Fatalf("expected 2 segments remaining, got %d", len(segs))
	}
	d, err := readDroppedSeq(dir)
	if err != nil {
		t.Fatal(err)
	}
	if d != 3 {
		t.Fatalf("expected droppedSeq 3, got %d", d)
	}
	// the 3 dropped files should no longer exist on disk
	for _, s := range segs {
		if s.id <= 3 {
			if _, err := os.Stat(s.path); !os.IsNotExist(err) {
				t.Fatalf("segment %d should have been removed", s.id)
			}
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestTruncateKeepsActiveSegment(t *testing.T) {
	dir := t.TempDir()
	w, err := openwal(Options{Dir: dir, SegmentMaxBytes: 30})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 5; i++ {
		if _, err := w.Write(TypeData, []byte("x"), [16]byte{}); err != nil {
			t.Fatal(err)
		}
	}
	// even though all records are <= seq 100, the active segment is never dropped
	if err := w.Truncate(100); err != nil {
		t.Fatal(err)
	}
	segs, _ := listSegments(dir)
	if len(segs) != 1 {
		t.Fatalf("expected only active segment, got %d", len(segs))
	}
	d, err := readDroppedSeq(dir)
	if err != nil {
		t.Fatal(err)
	}
	if d != 4 {
		t.Fatalf("expected droppedSeq 4 (prior segments), got %d", d)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	// reopen: lastSeq must still be 5 (4 dropped + 1 active record)
	w2, err := openwal(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	defer w2.Close()
	if w2.LastSeq() != 5 {
		t.Fatalf("expected lastSeq 5 after reopen, got %d", w2.LastSeq())
	}
}

func TestTruncateThenContinueSeq(t *testing.T) {
	dir := t.TempDir()
	w, err := openwal(Options{Dir: dir, SegmentMaxBytes: 30})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 5; i++ {
		if _, err := w.Write(TypeData, []byte("x"), [16]byte{}); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Truncate(3); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	w2, err := openwal(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	defer w2.Close()
	if w2.LastSeq() != 5 {
		t.Fatalf("expected lastSeq 5 after reopen, got %d", w2.LastSeq())
	}
	s, err := w2.Write(TypeData, []byte("x"), [16]byte{})
	if err != nil {
		t.Fatal(err)
	}
	if s != 6 {
		t.Fatalf("expected next seq 6, got %d", s)
	}
}

func TestTruncateAfterClose(t *testing.T) {
	dir := t.TempDir()
	w, err := openwal(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if err := w.Truncate(1); err != ErrClosed {
		t.Fatalf("expected ErrClosed, got %v", err)
	}
}

func TestTruncateIdempotentOnNoDrop(t *testing.T) {
	dir := t.TempDir()
	w, err := openwal(Options{Dir: dir, SegmentMaxBytes: 30})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		if _, err := w.Write(TypeData, []byte("x"), [16]byte{}); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Truncate(0); err != nil {
		t.Fatal(err)
	}
	if err := w.Truncate(0); err != nil {
		t.Fatalf("second Truncate(0) should be nil, got %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
}

// ---- meta file ----

func TestMetaRoundTrip(t *testing.T) {
	dir := t.TempDir()
	if err := writeDroppedSeq(dir, 42); err != nil {
		t.Fatal(err)
	}
	v, err := readDroppedSeq(dir)
	if err != nil {
		t.Fatal(err)
	}
	if v != 42 {
		t.Fatalf("expected 42, got %d", v)
	}
}

func TestReadDroppedSeqMissing(t *testing.T) {
	dir := t.TempDir()
	v, err := readDroppedSeq(dir)
	if err != nil {
		t.Fatal(err)
	}
	if v != 0 {
		t.Fatalf("expected 0 for missing meta, got %d", v)
	}
}

func TestReadDroppedSeqCorrupt(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(metaPath(dir), []byte{1, 2, 3}, 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := readDroppedSeq(dir)
	if err == nil {
		t.Fatal("expected error for corrupt (wrong-size) meta")
	}
}

// ---- listSegments ----

func TestListSegmentsIgnoresStrayFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "random.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "seg-00000000000000000007.db"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	// a segment with a non-numeric id should be ignored
	if err := os.WriteFile(filepath.Join(dir, "seg-abc.db"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	segs, err := listSegments(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(segs) != 1 || segs[0].id != 7 {
		t.Fatalf("expected only segment id 7, got %+v", segs)
	}
}

func TestListSegmentsSorted(t *testing.T) {
	dir := t.TempDir()
	for _, id := range []uint64{3, 1, 2} {
		if err := os.WriteFile(segmentPath(dir, id), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	segs, err := listSegments(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(segs) != 3 || segs[0].id != 1 || segs[1].id != 2 || segs[2].id != 3 {
		t.Fatalf("segments not sorted: %+v", segs)
	}
}

// ---- defaults ----

func TestOptionsDefaults(t *testing.T) {
	o := Options{}
	o.setDefaults()
	if o.SegmentMaxBytes != defaultSegMax {
		t.Fatalf("expected default segment size, got %d", o.SegmentMaxBytes)
	}
}

// ---- concurrency ----

func TestConcurrentWrites(t *testing.T) {
	dir := t.TempDir()
	w, err := openwal(Options{Dir: dir, SyncOnWrite: true})
	if err != nil {
		t.Fatal(err)
	}
	const n = 200
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := w.Write(TypeData, []byte("x"), [16]byte{}); err != nil {
				t.Errorf("write: %v", err)
			}
		}()
	}
	wg.Wait()
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if w.LastSeq() != uint64(n) {
		t.Fatalf("expected lastSeq %d, got %d", n, w.LastSeq())
	}
	_, seq := replayDir(t, dir)
	if seq != uint64(n) {
		t.Fatalf("expected %d records, got %d", n, seq)
	}
}

func TestConcurrentWritesAndTruncate(t *testing.T) {
	dir := t.TempDir()
	w, err := openwal(Options{Dir: dir, SegmentMaxBytes: 30})
	if err != nil {
		t.Fatal(err)
	}
	const n = 100
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w.Write(TypeData, []byte("x"), [16]byte{})
		}()
	}
	// concurrently truncate aggressively; must never panic or corrupt seq accounting
	go func() {
		for i := 0; i < 10; i++ {
			w.Truncate(uint64(i * 5))
		}
	}()
	wg.Wait()
	_ = w.Close()
	// reopen to confirm it recovers without error
	w2, err := openwal(Options{Dir: dir, SegmentMaxBytes: 30})
	if err != nil {
		t.Fatalf("reopen after concurrent truncate: %v", err)
	}
	_ = w2.Close()
}

// ---- error path: open bad dir ----

func TestOpenBadDir(t *testing.T) {
	// a file where we want a directory
	f := filepath.Join(t.TempDir(), "afile")
	if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := openwal(Options{Dir: filepath.Join(f, "sub")})
	if err == nil {
		t.Fatal("expected error opening WAL inside a file path")
	}
}

// ---- GetRecord offset correctness (bug #8) ----

func TestGetRecordReturnsCorrectPayload(t *testing.T) {
	dir := t.TempDir()
	w, err := openwal(Options{Dir: dir, SegmentMaxBytes: 1 << 20, SyncOnWrite: true})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	k1 := key16(1)
	k2 := key16(2)
	k3 := key16(3)

	if err := w.SetRecord(TypeData, []byte("alpha"), k1); err != nil {
		t.Fatal(err)
	}
	if err := w.SetRecord(TypeData, []byte("bravo"), k2); err != nil {
		t.Fatal(err)
	}
	if err := w.SetRecord(TypeData, []byte("charlie"), k3); err != nil {
		t.Fatal(err)
	}
	// re-set k2: GetRecord must return the LATEST write, which requires the
	// offset lookup to point at the record's start, not its end.
	if err := w.SetRecord(TypeData, []byte("bravo2"), k2); err != nil {
		t.Fatal(err)
	}

	if got, err := w.GetRecord(k1); err != nil || string(got) != "alpha" {
		t.Fatalf("k1: got %q err %v", got, err)
	}
	if got, err := w.GetRecord(k2); err != nil || string(got) != "bravo2" {
		t.Fatalf("k2: got %q err %v (expected latest value)", got, err)
	}
	if got, err := w.GetRecord(k3); err != nil || string(got) != "charlie" {
		t.Fatalf("k3: got %q err %v", got, err)
	}
	if _, err := w.GetRecord(key16(9)); err == nil {
		t.Fatal("expected error for a key that was never written")
	}
}

func TestGetRecordAfterReopen(t *testing.T) {
	dir := t.TempDir()
	w, err := openwal(Options{Dir: dir, SegmentMaxBytes: 1 << 20, SyncOnWrite: true})
	if err != nil {
		t.Fatal(err)
	}
	k1 := key16(1)
	k2 := key16(2)
	if err := w.SetRecord(TypeData, []byte("one"), k1); err != nil {
		t.Fatal(err)
	}
	if err := w.SetRecord(TypeData, []byte("two"), k2); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	// Reopen recovers the offset lookup by scanning segments; the stored offset
	// must be the record start so GetRecord can read it back.
	w2, err := openwal(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	defer w2.Close()

	if got, err := w2.GetRecord(k1); err != nil || string(got) != "one" {
		t.Fatalf("after reopen k1: got %q err %v", got, err)
	}
	if got, err := w2.GetRecord(k2); err != nil || string(got) != "two" {
		t.Fatalf("after reopen k2: got %q err %v", got, err)
	}
}

// ---- Initwal / expandHome (bugs #9 and #10) ----

func TestExpandHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home dir available")
	}
	got, err := expandHome("~/.config/supersand/db")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(home, ".config/supersand/db")
	if got != want {
		t.Fatalf("expandHome: got %q want %q", got, want)
	}
	if strings.HasPrefix(got, "~") {
		t.Fatal("expanded path must not start with ~")
	}
	// a path without a leading ~ is returned unchanged
	plain := "/tmp/foo/bar"
	if g, _ := expandHome(plain); g != plain {
		t.Fatalf("plain path should be unchanged, got %q", g)
	}
}

func TestInitwalWithTempDir(t *testing.T) {
	dir := t.TempDir()
	w := initwalWithOptions(dir)
	if w == nil {
		t.Fatal("expected non-nil WAL")
	}
	if w.active == nil {
		t.Fatal("active file should be set")
	}
	fi, err := os.Stat(w.active.Name())
	if err != nil {
		t.Fatalf("active file stat: %v", err)
	}
	if fi.IsDir() {
		t.Fatalf("active must be a regular segment file, not the directory %q", w.active.Name())
	}
	if !strings.HasSuffix(w.active.Name(), segSuffix) {
		t.Fatalf("active name should end with %q, got %q", segSuffix, w.active.Name())
	}
}

// Bug #9: the old fallback opened the directory path itself as a data file.
// When the configured dir is invalid, initwalWithOptions must NOT open it as a
// file — it should return nil after logging, not a WAL whose active handle is
// the directory/file path.
func TestInitwalFallbackDoesNotOpenDirAsFile(t *testing.T) {
	// Force openwal to fail by pointing "dir" at an existing regular file, so
	// MkdirAll (and thus openwal) cannot create a directory there.
	blocker := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	w := initwalWithOptions(blocker)
	if w != nil {
		t.Fatalf("expected nil WAL for invalid dir, got active=%v", w.active)
	}
}

// ---- WAL <-> bloom filter integration ----

func TestWalBloomFilterTracksWrites(t *testing.T) {
	dir := t.TempDir()
	w, err := openwal(Options{Dir: dir, SegmentMaxBytes: 1 << 20, SyncOnWrite: true})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	k1 := key16(1)
	if err := w.SetRecord(TypeData, []byte("v1"), k1); err != nil {
		t.Fatal(err)
	}
	if got, err := w.GetRecord(k1); err != nil || string(got) != "v1" {
		t.Fatalf("present key: got %q err %v", got, err)
	}
	// never-written key must be reported absent (caught by the bloom filter,
	// avoiding a disk seek)
	if _, err := w.GetRecord(key16(99)); err == nil {
		t.Fatal("absent key should report 'no record' via bloom filter")
	}
}

func TestWalBloomFilterRecoversAcrossReopen(t *testing.T) {
	dir := t.TempDir()
	w, err := openwal(Options{Dir: dir, SegmentMaxBytes: 1 << 20, SyncOnWrite: true})
	if err != nil {
		t.Fatal(err)
	}
	k1 := key16(1)
	k2 := key16(2)
	if err := w.SetRecord(TypeData, []byte("one"), k1); err != nil {
		t.Fatal(err)
	}
	if err := w.SetRecord(TypeData, []byte("two"), k2); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	// Reopen: the bloom filter must be rebuilt by scanning segments so that
	// GetRecord still finds the previously-written keys.
	w2, err := openwal(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	defer w2.Close()

	if got, err := w2.GetRecord(k1); err != nil || string(got) != "one" {
		t.Fatalf("recovered k1: got %q err %v", got, err)
	}
	if got, err := w2.GetRecord(k2); err != nil || string(got) != "two" {
		t.Fatalf("recovered k2: got %q err %v", got, err)
	}
	// a key never written must still be reported absent
	if _, err := w2.GetRecord(key16(7)); err == nil {
		t.Fatal("absent key should report 'no record' after reopen")
	}
}

// A deleted (tombstoned) key must still be present in the bloom filter, so
// GetRecord reaches the delete branch ("record is deleted") instead of being
// short-circuited as "no record for this key".
func TestWalBloomFilterIncludesDeletedKey(t *testing.T) {
	dir := t.TempDir()
	w, err := openwal(Options{Dir: dir, SegmentMaxBytes: 1 << 20, SyncOnWrite: true})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	k1 := key16(1)
	if err := w.SetRecord(TypeData, []byte("v"), k1); err != nil {
		t.Fatal(err)
	}
	if err := w.SetRecord(TypeDelete, nil, k1); err != nil {
		t.Fatal(err)
	}
	_, err = w.GetRecord(k1)
	if err == nil {
		t.Fatal("expected an error for a deleted key")
	}
	if err.Error() != "record is deleted" {
		t.Fatalf("expected 'record is deleted', got %q", err.Error())
	}
}
