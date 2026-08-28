package store

import(
  "sort"
	"strconv"
	"strings"
  "hash/crc32"
  "bufio"
	"encoding/binary"
  "os"
	"path/filepath"
  "fmt"
)


func(w *WAL) SetRecord(t RecordType, payload []byte,key [16]byte)error{
   _,err:=w.Write(t,payload,key)
	 
	 if err !=nil{
		 return fmt.Errorf("error in SetRecord:%w",err)
	 }
	 w.offsetlookup[key]=w.lastoffset
	 return nil
}

// Write appends a record and returns its assigned sequence number.
// The payload is copied is NOT retained by the WAL — reuse the slice freely
// after Write returns.

// i am changing the record format now the headr will store crc key offset tpe len. after the payload !importan what is the sequence number is it offset. no it is not offset ther is nothing that track the offset.

func (w *WAL) Write(t RecordType, payload []byte,key [16]byte) (uint64, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return 0, ErrClosed
	}

	recLen := headerSize 	
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

	seq := w.lastSeq + 1 // incrsing lastSeq


	var hdr [headerSize]byte
	crc := crc32.NewIEEE()
	crc.Write([]byte{byte(t)})
	crc.Write(payload)
	sum := crc.Sum32()

	binary.LittleEndian.PutUint32(hdr[0:4], sum)
	hdr[4] = byte(t)
	binary.LittleEndian.PutUint32(hdr[5:7],uint32(len(key)))
	binary.LittleEndian.PutUint32(hdr[7:11], uint32(len(payload)))
	var w1 int
	var w2 int
	var w3 int
	 w1, err := w.bw.Write(hdr[:])
	 if err != nil {
		return 0, fmt.Errorf("wal: write header: %w", err)
	}
	if len(key)>0{
		w2,err=w.bw.Write(key)
		
		if err!=nil{
			return 0,fmt.Errorf("wal:write error at key write:%w",err)
		}
	}
	if len(payload) > 0 {
		 w3, err = w.bw.Write(payload)
		 if err != nil {
			return 0, fmt.Errorf("wal: write payload: %w", err)
		}
	}

	w.curSize += int64(recLen)
	w.lastSeq = seq
	w.lastoffset += uint64(w1+w2+w3)

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


