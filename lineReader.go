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
}

// NewLineReader makes a new LineReader from a Reader.
func NewLineReader(reader io.Reader) *LineReader {
	return &LineReader{bufio.NewReader(reader), 0, newBuffer()}
}

// NewLineReaderAtPosition makes a line scanner that will start scanning at a specific position within the given file.
func NewLineReaderAtPosition(source io.ReadSeeker, position int64) (*LineReader, error) {
	offset, err := source.Seek(position, io.SeekStart)
	if err != nil {
		return nil, err
	}
	if offset != position {
		return nil, errors.New("cant reposition within file")
	}
	ls := &LineReader{bufio.NewReader(source), position, newBuffer()}
	return ls, nil
}

// ReadLine reads the next line from the input.
// Returns the scanned line, the position within the file where the next line will start and any occurring error.
// Either a line is matched, in which case (line, position, nil) is returned, or an error is found and
// (nil, 0, err) is returned.
//
// Unlike bufio.Scanner, it treats EOF differently. Here, EOF is expected to be transient since writers may be appending
// content intermitently. When a partial line is read and EOF is reached before finding a newline, the line is not
// returned, and io.EOF will. Subsequent calls to ReadLine() may finish scanning the line if a newline is ultimately
// appended.
// Newline is either '\n' or '\r\n'. Lines are returned without newline and they can be empty.
//
// For force reading the partially unterminated line at the end of the file, call ReadLastLine()
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
				return nil, 0, errors.New("ReadSlice found newLine but returned empty")
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
		if readError == io.EOF {
			return nil, 0, io.EOF
		}
		return nil, 0, readError
	}
}

// ReadLastLine reads the next line, even if it's unterminated.
// This has the same behavior ar bufio.Scanner.Scan() when configured with bufio.ScanLines
func (lr *LineReader) ReadLastLine() ([]byte, error) {
	line, _, err := lr.ReadLine()
	if err == io.EOF {
		line = lr.buffer // note that we don't dropCR from the buffer since we haven't found a newline.
		lr.buffer = newBuffer()
		return line, nil
	}
	return line, err
}

const DEFAULT_LINE_BUFFER_SIZE = 4096

// newBuffer makes a new, empty buffer for the next line to be read
func newBuffer() []byte {
	return make([]byte, 0, DEFAULT_LINE_BUFFER_SIZE)
}

// dropCR drops a terminal \r from a newline terminated line.
func dropCR(line []byte) []byte {
	if len(line) > 0 && line[len(line)-1] == '\r' {
		return line[0 : len(line)-1]
	}
	return line
}
