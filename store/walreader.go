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
 
