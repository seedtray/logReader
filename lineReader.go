package logReader

import (
	"bufio"
	"errors"
	"io"
)

// A LineReader will read full lines from a Reader, typically a file being appended by an external program.
// This provides similar functionality to bufio.Scanner when using bufio.ScanLines as a splitter. See ReadLine
// for the differences.
type LineReader struct {
	reader       *bufio.Reader
	nextPosition int64
	buffer       []byte
	finalized    bool
}

// NewLineReader makes a new LineReader from a Reader.
func NewLineReader(reader io.Reader) *LineReader {
	return &LineReader{bufio.NewReader(reader), 0, newBuffer(), false}
}

// NewLineReaderAtPosition makes a line scanner that will start scanning at a specific position within the given file.
// fileFinalized tells the file is not expected to be further appended to. See ReadLine
func NewLineReaderAtPosition(source io.ReadSeeker, position int64, fileFinalized bool) (*LineReader, error) {
	offset, err := source.Seek(position, io.SeekStart)
	if err != nil {
		return nil, err
	}
	if offset != position {
		return nil, errors.New("cant reposition within file")
	}
	lineReader := &LineReader{reader: bufio.NewReader(source), nextPosition: position, buffer: newBuffer(), finalized: fileFinalized}
	return lineReader, nil
}

// ReadLine reads the next line from the input.
// Returns the scanned line, the position within the file where the next line will start and any occurring error.
// Typically, either a line is matched, in which case (line, position, nil) is returned, or an error is found and
// (nil, 0, err) is returned. The exception is around EOF, unterminated lines and finalized files:
//
// After a LineReader consider its source file finalized, it behaves like bufio.ScanLines. This means
// that a possibly unterminated line at the end of the file will be considered complete, and returned (along with its
// position). In that case the 3 values (line, position and error) will be non nil and meaningful.
//
// On the contrary, when a LineReader considers its source file as not finalized, unterminated lines are not returned,
// in the expectation that the file will be appended to externally and the line will be properly terminated.
// In this case, ReadLine returns (nil, 0, io.EOF)
//
// Newline is either '\n' or '\r\n'. Lines are returned without newline, and they can be empty.
//
func (lr *LineReader) ReadLine() ([]byte, int64, error) {
	for {
		read, readError := lr.reader.ReadSlice('\n')
		if len(read) > 0 {
			lr.buffer = append(lr.buffer, read...)
			lr.nextPosition += int64(len(read))
		}
		if readError == nil {
			if len(read) == 0 {
				panic("ReadSlice found newLine but returned empty")
			}
			line := dropCR(lr.buffer[:len(lr.buffer)-1])
			lr.buffer = newBuffer()
			return line, lr.nextPosition, nil
		}
		if readError == bufio.ErrBufferFull {
			// we already got the entire buffer as a read fragment and put it in our buffer, so
			// we're good to carry on
			continue
		}
		if lr.finalized && readError == io.EOF && len(lr.buffer) > 0 {
			line := lr.buffer
			lr.buffer = nil
			return line, lr.nextPosition, readError
		}
		return nil, 0, readError
	}
}

// Finalize considers the underlying file being read by LineReader as finalized, meaning no further appends are
// expected on it. This affects how EOF is treated by ReadLine.
func (lr *LineReader) Finalize() {
	lr.finalized = true
}

const DefaultLineBufferSize = 4096

// newBuffer makes a new, empty buffer for the next line to be read
func newBuffer() []byte {
	return make([]byte, 0, DefaultLineBufferSize)
}

// dropCR drops a terminal \r from a newline terminated line.
func dropCR(line []byte) []byte {
	if len(line) > 0 && line[len(line)-1] == '\r' {
		return line[0 : len(line)-1]
	}
	return line
}
