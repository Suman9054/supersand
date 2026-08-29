package store

import(
 "bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	)


func (w *WAL)GetRecord(key [16]byte)(payload []byte,err error){
	var def []byte
  // The bloom filter can only report false positives, never false negatives,
  // so a negative answer means the key was never written and we can skip the
  // disk seek entirely.
  if w.filter != nil && !w.filter.test(key[:]) {
		return def,fmt.Errorf("no record for this key")
	}
  offset,ok:=w.offsetlookup[key]
	if !ok {
		return def,fmt.Errorf("no record for this key")
	}
	_,serr:=w.active.Seek(offset,io.SeekStart)
  if serr != nil {
		return def,fmt.Errorf("failed to seek at file")
	}
 bf:=bufio.NewReaderSize(w.active,64*1024)
  
  record,_,err:=readRecord(bf)
	if err != nil {
		return def,err
	}

	if record.Type == TypeDelete {
		return def,fmt.Errorf("record is deleted")
	}
 return record.Payload,nil
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



func scanSegment(path string) (recordCount uint64, validEnd int64,scanofsetmap map[[16]byte]int64, err error) {
	var defultap map[[16]byte]int64
	f, err := os.Open(path)
	if err != nil {
		return 0, 0, defultap,fmt.Errorf("wal: scan open: %w", err)
	}
	defer f.Close()
 
	br := bufio.NewReaderSize(f, 64*1024)
	var offset int64
	var count uint64
  offsetlookup:=make(map[[16]byte]int64)
	for {
		before := offset
		rec, n, err := readRecord(br)
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
			return 0, 0, defultap, err
		}
		count++
		offset = before + int64(n)
		// Store the start of the record (not its end) so GetRecord can seek
		// straight to it and read it back.
		offsetlookup[rec.Key] = before
	}
 
	return count, offset, offsetlookup,nil
}




// readRecord reads one record from br. Returns io.EOF at a clean segment
// boundary, ErrCorrupt if the checksum fails or the file ends mid-record.
// The returned int is the exact number of bytes the record occupies on disk
// (header + key + payload), used by callers to advance their byte offset.
func readRecord(br *bufio.Reader) (Record, int, error) {
	var hdr [headerSize]byte
	n, err := io.ReadFull(br, hdr[:])
	if err == io.EOF && n == 0 {
		return Record{}, 0, io.EOF
	}
	if err != nil {
		// Partial header at EOF = torn write from a crash mid-append.
		// Treat as end of valid data, not a hard error, so replay can stop cleanly.
		if errors.Is(err, io.ErrUnexpectedEOF) {
			return Record{}, 0, io.EOF
		}
		return Record{}, 0, fmt.Errorf("wal: read header: %w", err)
	}
 
	wantCRC := binary.LittleEndian.Uint32(hdr[0:4])
	typ := RecordType(hdr[4])
	keylen := binary.LittleEndian.Uint16(hdr[5:7])
	payloadLen := binary.LittleEndian.Uint32(hdr[7:11])
 
	payload := make([]byte, payloadLen)
	key := make([]byte, keylen)
	if keylen > 0 {
		if _, err := io.ReadFull(br, key); err != nil {
			if errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF) {
				return Record{}, 0, io.EOF // torn write: header written, key truncated
			}
			return Record{}, 0, fmt.Errorf("wal: read key: %w", err)
		}
	}
	if payloadLen > 0 {
		if _, err := io.ReadFull(br, payload); err != nil {
			if errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF) {
				return Record{}, 0, io.EOF // torn write: header written, payload truncated
			}
			return Record{}, 0, fmt.Errorf("wal: read payload: %w", err)
		}
	}
 
	crc := crc32.NewIEEE()
	crc.Write([]byte{byte(typ)})
	crc.Write(payload)
	if crc.Sum32() != wantCRC {
		return Record{}, 0, ErrCorrupt
	}

	var keyarray [16]byte
	copy(keyarray[:], key)
	recBytes := headerSize + int(keylen) + int(payloadLen)
	return Record{Key: keyarray, Type: typ, Payload: payload}, recBytes, nil
}
 
