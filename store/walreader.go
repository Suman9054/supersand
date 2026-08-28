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


func (w *WAL)GetRecord(key [16]byte)[]byte{

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
			return 0, 0, defultap,err
		}
		count++
		offset = before + int64(headerSize+len(rec.Payload))
		offsetlookup[rec.Key]=offset
	}
 
	return count, offset, offsetlookup,nil
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
	typ := RecordType(hdr[4])
  keylen:=binary.LittleEndian.Uint32(hdr[5:7])
	payloadLen:=binary.LittleEndian.Uint32(hdr[7:11])
 
	payload := make([]byte, payloadLen)
	key :=make([]byte,keylen)
	if payloadLen > 0 && keylen > 0{

		if _, err := io.ReadFull(br, key); err != nil {
			if errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF) {
				return Record{}, io.EOF // torn write: header written, payload truncated
			}
			return Record{}, fmt.Errorf("wal: read payload: %w", err)
		}

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

	var keyarray [16]byte
  copy(keyarray[:],key)
	return Record{Key:keyarray,Type: typ, Payload: payload}, nil
}
 
