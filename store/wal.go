// Record format on disk :
//
//	+----------+----------------+----+-----------------------+---------------+--------+
//	| CRC (4B) | Len (Uvarient) | Type(1B) | Payload (Len B) |index(Uvarient)|offset  |
//	+----------+----------------+----+-----------------------+---------------+--------+
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
	"io"
	"os"
	"path/filepath"
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
	Offset  uint64
	key     []byte
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

	// so of offset trake the cureent of set it will be set an offset at time 
  // of every write i will do offset = offset + uint64(record)
  lastoffset  uint64

 // closed is to ensure no race condition on file write 
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



