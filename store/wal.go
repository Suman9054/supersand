// Record format on disk :
//
//	+----------+----------+----------+-----------------+
//	| CRC (4B) | Len (4B) | Type(1B) | Payload (Len B) |
//	+----------+----------+----------+-----------------+
//
// CRC is crc32.Checksum over (Type || Payload).
// Records are grouped into fixed-size segment files so old segments
// can be truncated/archived independently of the active one.
package store

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
)

const (
	headerSize    = 9 // crc(4) + len(4) + type(1)
	defaultSegMax = 64 * 1024 * 1024 // 64MB per segment
	segPrefix     = "seg-"
	segSuffix     = ".wal"
)

// RecordType  SET vs DELETE
// without parsing the payload itself.
type RecordType uint8

const (
	TypeData RecordType = iota
	TypeCommit
	TypeCheckpoint
)

var (
	ErrCorrupt  = errors.New("wal: corrupt record")
	ErrClosed   = errors.New("wal: closed")
	ErrTooLarge = errors.New("wal: record exceeds segment size")
)

// Record is a single WAL entry with its assigned sequence number.
type Record struct {
	Seq     uint64
	Type    RecordType
	Payload []byte
}

// Options  for configures the WAL.
type Options struct {
	// Dir is the directory holding segment files. Created if missing.
	Dir string
	// SegmentMaxBytes caps each segment file's size before rolling to a new one.
	SegmentMaxBytes int64
	// SyncOnWrite fsyncs after every Write. Slower, but durable per-record.
	// If false, call Sync() explicitly (e.g. on a timer or batch boundary).
	SyncOnWrite bool
}

func (o *Options) setDefaults() {
	if o.SegmentMaxBytes <= 0 {
		o.SegmentMaxBytes = defaultSegMax
	}
}

// WAL is a segmented, append-only write-ahead log.
// Safe for concurrent use.
type WAL struct {
	mu   sync.Mutex
	opts Options

	segments []segmentMeta // sorted ascending by id
	active   *os.File
	bw       *bufio.Writer
	curSize  int64
	curID    uint64

	lastSeq uint64
	// droppedSeq is the global sequence number of the last record ever
	// removed by Truncate. New Readers seed their sequence counting from
	// here instead of assuming segments[0] starts at seq 1 — after a
	// Truncate, it usually doesn't. Persisted to metaPath so it survives
	// a process restart, since the segments that would let us recompute
	// it by scanning are exactly the ones Truncate deleted.
	droppedSeq uint64

	closed bool
}

const metaFileName = "meta"

func metaPath(dir string) string {
	return filepath.Join(dir, metaFileName)
}

func readDroppedSeq(dir string) (uint64, error) {
	data, err := os.ReadFile(metaPath(dir))
	if os.IsNotExist(err) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("wal: read meta: %w", err)
	}
	if len(data) != 8 {
		return 0, fmt.Errorf("wal: meta file corrupt (want 8 bytes, got %d)", len(data))
	}
	return binary.LittleEndian.Uint64(data), nil
}

// writeDroppedSeq persists atomically via write-temp-then-rename so a crash
// mid-write can't leave a torn meta file (same hazard as segment torn
// tails, different mechanism since this file has no per-record checksum).
func writeDroppedSeq(dir string, seq uint64) error {
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], seq)
	tmp := metaPath(dir) + ".tmp"
	if err := os.WriteFile(tmp, buf[:], 0o644); err != nil {
		return fmt.Errorf("wal: write meta tmp: %w", err)
	}
	if err := os.Rename(tmp, metaPath(dir)); err != nil {
		return fmt.Errorf("wal: rename meta: %w", err)
	}
	// fsync the directory entry too, or the rename itself isn't durable
	// across a crash on some filesystems.
	d, err := os.Open(dir)
	if err != nil {
		return fmt.Errorf("wal: open dir for fsync: %w", err)
	}
	defer d.Close()
	return d.Sync()
}

type segmentMeta struct {
	id   uint64
	path string
}

// Open creates or reopens a WAL at opts.Dir. It scans existing segments to
// recover the last sequence number, then appends to (or creates) the newest
// segment.
func Open(opts Options) (*WAL, error) {
	opts.setDefaults()
	if err := os.MkdirAll(opts.Dir, 0o755); err != nil {
		return nil, fmt.Errorf("wal: mkdir: %w", err)
	}

	w := &WAL{opts: opts}

	dropped, err := readDroppedSeq(opts.Dir)
	if err != nil {
		return nil, err
	}
	w.droppedSeq = dropped

	segs, err := listSegments(opts.Dir)
	if err != nil {
		return nil, err
	}
	w.segments = segs

	if len(segs) == 0 {
		w.lastSeq = dropped // nothing on disk, but Truncate may have run before every remaining segment was somehow also removed (e.g. all data truncated) — don't let seq numbering restart from 0
		if err := w.rollSegment(1); err != nil {
			return nil, err
		}
	} else {
		// Recover global lastSeq starting from droppedSeq (records Truncate
		// already removed still count toward the sequence), then counting
		// records in every remaining prior segment (assumed fully valid —
		// only the last segment can have a torn tail from a crash mid-write)
		// plus the valid prefix of the last segment.
		total := dropped
		for _, s := range segs[:len(segs)-1] {
			n, _, err := scanSegment(s.path)
			if err != nil {
				return nil, err
			}
			total += n
		}

		last := segs[len(segs)-1]
		w.curID = last.id
		f, err := os.OpenFile(last.path, os.O_RDWR|os.O_APPEND, 0o644)
		if err != nil {
			return nil, fmt.Errorf("wal: open last segment: %w", err)
		}
		lastCount, validEnd, err := scanSegment(last.path)
		if err != nil {
			f.Close()
			return nil, err
		}
		w.lastSeq = total + lastCount
		if err := f.Truncate(validEnd); err != nil {
			f.Close()
			return nil, fmt.Errorf("wal: truncate torn tail: %w", err)
		}
		if _, err := f.Seek(0, io.SeekEnd); err != nil {
			f.Close()
			return nil, err
		}
		w.active = f
		w.bw = bufio.NewWriterSize(f, 64*1024)
		w.curSize = validEnd
	}

	return w, nil
}

// Write appends a record and returns its assigned sequence number.
// The payload is copied is NOT retained by the WAL — reuse the slice freely
// after Write returns.
func (w *WAL) Write(t RecordType, payload []byte) (uint64, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return 0, ErrClosed
	}

	recLen := headerSize + len(payload)
	if int64(recLen) > w.opts.SegmentMaxBytes {
		return 0, ErrTooLarge
	}

	if w.curSize+int64(recLen) > w.opts.SegmentMaxBytes {
		if err := w.flushLocked(); err != nil {
			return 0, err
		}
		if err := w.rollSegment(w.curID + 1); err != nil {
			return 0, err
		}
	}

	seq := w.lastSeq + 1

	var hdr [headerSize]byte
	crc := crc32.NewIEEE()
	crc.Write([]byte{byte(t)})
	crc.Write(payload)
	sum := crc.Sum32()

	binary.LittleEndian.PutUint32(hdr[0:4], sum)
	binary.LittleEndian.PutUint32(hdr[4:8], uint32(len(payload)))
	hdr[8] = byte(t)

	if _, err := w.bw.Write(hdr[:]); err != nil {
		return 0, fmt.Errorf("wal: write header: %w", err)
	}
	if len(payload) > 0 {
		if _, err := w.bw.Write(payload); err != nil {
			return 0, fmt.Errorf("wal: write payload: %w", err)
		}
	}

	w.curSize += int64(recLen)
	w.lastSeq = seq

	if w.opts.SyncOnWrite {
		if err := w.flushLocked(); err != nil {
			return 0, err
		}
		if err := w.active.Sync(); err != nil {
			return 0, fmt.Errorf("wal: fsync: %w", err)
		}
	}

	return seq, nil
}

// Sync flushes buffered writes and fsyncs the active segment.
// Call this on your own cadence (batch boundary, timer) if SyncOnWrite=false.
func (w *WAL) Sync() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return ErrClosed
	}
	if err := w.flushLocked(); err != nil {
		return err
	}
	return w.active.Sync()
}

func (w *WAL) flushLocked() error {
	if err := w.bw.Flush(); err != nil {
		return fmt.Errorf("wal: flush: %w", err)
	}
	return nil
}

// LastSeq returns the most recently assigned sequence number (0 if empty).
func (w *WAL) LastSeq() uint64 {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.lastSeq
}

// Close flushes, fsyncs, and closes the active segment.
func (w *WAL) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return nil
	}
	w.closed = true
	if err := w.flushLocked(); err != nil {
		w.active.Close()
		return err
	}
	if err := w.active.Sync(); err != nil {
		w.active.Close()
		return err
	}
	return w.active.Close()
}

func (w *WAL) rollSegment(id uint64) error {
	if w.active != nil {
		w.active.Close()
	}
	path := segmentPath(w.opts.Dir, id)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return fmt.Errorf("wal: create segment %d: %w", id, err)
	}
	w.active = f
	w.bw = bufio.NewWriterSize(f, 64*1024)
	w.curSize = 0
	w.curID = id
	w.segments = append(w.segments, segmentMeta{id: id, path: path})
	return nil
}

// Truncate removes all segments whose highest sequence number is <= seq,
// i.e. everything already checkpointed/applied. Keeps at least the active
// segment. Safe to call while the WAL is open for writes.
func (w *WAL) Truncate(seq uint64) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return ErrClosed
	}

	keep := w.segments[len(w.segments)-1:] // never drop the active segment
	var drop []segmentMeta
	runningTotal := w.droppedSeq // global seq accounting must continue from what's already dropped, not restart at 0
	var newDroppedSeq uint64 = w.droppedSeq
	for _, s := range w.segments[:len(w.segments)-1] {
		n, _, err := scanSegment(s.path)
		if err != nil {
			return err
		}
		runningTotal += n // global seq of the last record in segment s
		if runningTotal <= seq {
			drop = append(drop, s)
			newDroppedSeq = runningTotal
		} else {
			keep = append([]segmentMeta{s}, keep...)
		}
	}

	if len(drop) == 0 {
		return nil // nothing to do; avoid a pointless meta write+fsync
	}

	// Persist the new droppedSeq BEFORE removing any files, and fsync it.
	// If we crash after this point but before all removals finish, the
	// next Open() sees the correct (advanced) droppedSeq and any segment
	// files still physically present just get re-scanned as redundant —
	// harmless. The reverse ordering (remove-then-persist) would let a
	// crash mid-way lose the count for segments already deleted.
	if err := writeDroppedSeq(w.opts.Dir, newDroppedSeq); err != nil {
		return err
	}
	w.droppedSeq = newDroppedSeq

	for _, s := range drop {
		if err := os.Remove(s.path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("wal: remove segment %d: %w", s.id, err)
		}
	}
	sort.Slice(keep, func(i, j int) bool { return keep[i].id < keep[j].id })
	w.segments = keep
	return nil
}

func segmentPath(dir string, id uint64) string {
	return filepath.Join(dir, fmt.Sprintf("%s%020d%s", segPrefix, id, segSuffix))
}

func listSegments(dir string) ([]segmentMeta, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("wal: read dir: %w", err)
	}
	var segs []segmentMeta
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasPrefix(name, segPrefix) || !strings.HasSuffix(name, segSuffix) {
			continue
		}
		idStr := strings.TrimSuffix(strings.TrimPrefix(name, segPrefix), segSuffix)
		id, err := strconv.ParseUint(idStr, 10, 64)
		if err != nil {
			continue // ignore stray files
		}
		segs = append(segs, segmentMeta{id: id, path: filepath.Join(dir, name)})
	}
	sort.Slice(segs, func(i, j int) bool { return segs[i].id < segs[j].id })
	return segs, nil
}


// scanSegment reads a segment start-to-end and returns the number of valid
// records in it plus the byte offset where valid data ends. A torn tail
// (partial header/payload from a crash mid-write, or a bad checksum) stops
// the scan and is excluded from both the count and validEnd — callers use
// validEnd to truncate the file back to its last known-good record.
//
// Counts are local to this segment. wal.go's Open() sums counts across all
// prior segments (plus droppedSeq) to recover the true global lastSeq, and
// Truncate() sums them the same way to find each segment's global high-water
// mark before deciding whether it's fully covered and safe to delete.



func scanSegment(path string) (recordCount uint64, validEnd int64, err error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, 0, fmt.Errorf("wal: scan open: %w", err)
	}
	defer f.Close()
 
	br := bufio.NewReaderSize(f, 64*1024)
	var offset int64
	var count uint64
 
	for {
		before := offset
		rec, err := readRecord(br)
		if err == io.EOF {
			break
		}
		if err == ErrCorrupt {
			// Stop at first corruption; everything before `before` is valid.
			// This matches WAL semantics: a corrupt/torn tail is discarded,
			// not treated as fatal, since it can only be the very last write.
			break
		}
		if err != nil {
			return 0, 0, err
		}
		count++
		offset = before + int64(headerSize+len(rec.Payload))
	}
 
	return count, offset, nil
}




// readRecord reads one record from br. Returns io.EOF at a clean segment
// boundary, ErrCorrupt if the checksum fails or the file ends mid-record.
func readRecord(br *bufio.Reader) (Record, error) {
	var hdr [headerSize]byte
	n, err := io.ReadFull(br, hdr[:])
	if err == io.EOF && n == 0 {
		return Record{}, io.EOF
	}
	if err != nil {
		// Partial header at EOF = torn write from a crash mid-append.
		// Treat as end of valid data, not a hard error, so replay can stop cleanly.
		if errors.Is(err, io.ErrUnexpectedEOF) {
			return Record{}, io.EOF
		}
		return Record{}, fmt.Errorf("wal: read header: %w", err)
	}
 
	wantCRC := binary.LittleEndian.Uint32(hdr[0:4])
	payloadLen := binary.LittleEndian.Uint32(hdr[4:8])
	typ := RecordType(hdr[8])
 
	payload := make([]byte, payloadLen)
	if payloadLen > 0 {
		if _, err := io.ReadFull(br, payload); err != nil {
			if errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF) {
				return Record{}, io.EOF // torn write: header written, payload truncated
			}
			return Record{}, fmt.Errorf("wal: read payload: %w", err)
		}
	}
 
	crc := crc32.NewIEEE()
	crc.Write([]byte{byte(typ)})
	crc.Write(payload)
	if crc.Sum32() != wantCRC {
		return Record{}, ErrCorrupt
	}
 
	return Record{Type: typ, Payload: payload}, nil
}
 
