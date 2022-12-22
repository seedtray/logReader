package logReader

import (
	"bufio"
	"fmt"
	"github.com/spf13/afero"
	"io"
	"strings"
	"syscall"
	"testing"
	"testing/iotest"
)

var TestFs = afero.NewMemMapFs()

type FileForTest struct {
	t      *testing.T
	Reader io.ReadSeeker
	Writer afero.File
}

// Writes a string at the end of a file, ensuring there were no errors while doing so.
func (ft *FileForTest) Append(s string) {
	_, err := fmt.Fprintf(ft.Writer, s)
	if err != nil {
		ft.t.Error(err)
	}
}

// Makes a new FileForTest using the test name as the filename.
func newFileForTest(t *testing.T) *FileForTest {
	rw, err := TestFs.OpenFile(t.Name(), syscall.O_CREAT|syscall.O_APPEND|syscall.O_SYNC, 0600)
	if err != nil {
		t.Error(err)
	}
	ro, err := TestFs.Open(t.Name())
	if err != nil {
		t.Error(err)
	}
	return &FileForTest{t, ro, rw}
}

// Ensures that a reader hits EOF when calling ReadLine.
// It also ensures that ReadLine doesn't return any error.
func assertReadLineReachesEOF(t *testing.T, lr *LineReader) {
	t.Helper()
	line, _, err := lr.ReadLine()
	if err != io.EOF {
		t.Error(err)
	}
	if line != nil {
		t.Errorf("reader found an unexpected line: '%s'", line)
	}
}

// Ensures that calling ReadLine() once will find the expected line
// Note that lines are typically []byte, but we're using strings here for convenience.
func assertReadLineFindsLine(t *testing.T, lr *LineReader, expected string) {
	t.Helper()
	line, _, err := lr.ReadLine()
	if err != nil && err != io.EOF {
		t.Error(err)
	}
	if string(line) != expected {
		t.Errorf("Expected to find line:\n%s\nBut found\n%s", expected, line)
	}
}

// Ensures that LineReader would find the expected lines.
// It doesn't continue scanning after the last matched line.
func assertReadLineFindsLines(t *testing.T, lr *LineReader, expected []string) {
	t.Helper()
	for _, line := range expected {
		assertReadLineFindsLine(t, lr, line)
	}
}

// Test that calling ReadLine() on an empty file will just hit EOF
func TestEmptyFileReturnsEOF(t *testing.T) {
	reader := NewLineReader(iotest.ErrReader(io.EOF))
	assertReadLineReachesEOF(t, reader)
}

// Test that calling ReadLine() on a file with a single and terminated line will find it.
func TestSingleLineIsFound(t *testing.T) {
	reader := NewLineReader(strings.NewReader("Hello World\n"))
	assertReadLineFindsLine(t, reader, "Hello World")
}

// Test that a file holding two terminated lines will be found by calling ReadLine() repeatedly.
func TestTwoLinesAreFound(t *testing.T) {
	tf := newFileForTest(t)
	reader := NewLineReader(tf.Reader)
	tf.Append("Hello world\nAnother\n")
	assertReadLineFindsLines(t, reader, []string{"Hello world", "Another"})
}

// Test that an unterminated line at the end of a file will not be returned by ReadLine,
// and ReadLastLine will return it
func TestUnterminatedLineAtEOFNotReadImplicitly(t *testing.T) {
	tf := newFileForTest(t)
	reader := NewLineReader(tf.Reader)
	tf.Append("Hello world\nUnterminated")
	assertReadLineFindsLines(t, reader, []string{"Hello world"})
	reader.Finalize()
	assertReadLineFindsLines(t, reader, []string{"Unterminated"})
}

// Test that we don't remove a trailing \r from an unterminated line read by ReadLastLine()
func TestCRNotRemovedFromUnterminatedLine(t *testing.T) {
	tf := newFileForTest(t)
	reader := NewLineReader(tf.Reader)
	assertReadLineFindsLines(t, reader, nil)
	tf.Append("Unterminated\r")
	reader.Finalize()
	assertReadLineFindsLines(t, reader, []string{"Unterminated\r"})
}

// A reader han hit eof, but if the file gets written by a third party, it will find those
//appended contents on the next call to ReadLine()
func TestCanReadAppendsAfterReachingEOF(t *testing.T) {
	ft := newFileForTest(t)
	reader := NewLineReader(ft.Reader)

	ft.Append("Hello ")
	assertReadLineReachesEOF(t, reader)

	ft.Append("World\n")
	assertReadLineFindsLine(t, reader, "Hello World")
}

// A LineReader will not read a line unless it's finished by a newline, even at EOF.
// This is relevant because a half written line at the end of the file is ambiguous:
// It might be the last (but unterminated!) full line of a file, or it's a partially written line.
// Typically, the reader would need more context in order to determine this.
// For example, a multiple file reader would consider a file closed (and immutable) after the
// next one is found, hence disambiguating that last fragment as a whole line.
func TestWaitsForNewlineAtEOF(t *testing.T) {
	ft := newFileForTest(t)
	reader := NewLineReader(ft.Reader)

	ft.Append("Hello World\nHey!")
	assertReadLineFindsLine(t, reader, "Hello World")
	assertReadLineReachesEOF(t, reader)
	ft.Append("\n")
	assertReadLineFindsLine(t, reader, "Hey!")
}

// helper function to create a string full of zeroes.
func makeZeroesString(length uint) string {
	return fmt.Sprintf("%*d", length, 0)
}

// Test that the if the underlying file is served by a buffered reader,
// LineReader can find lines larger than the buffer size.
func TestLineCanBeBiggerThanBufferSize(t *testing.T) {
	contents := makeZeroesString(1000)
	ft := newFileForTest(t)
	ft.Append(contents + "\n")
	reader := NewLineReader(nil)
	reader.reader = bufio.NewReaderSize(ft.Reader, 5)
	assertReadLineFindsLine(t, reader, contents)
}

func TestCanResumeAfterOneLineRead(t *testing.T) {
	ft := newFileForTest(t)
	ft.Append("line1\nline2\nline3\n")
	reader := NewLineReader(ft.Reader)
	line, position, err := reader.ReadLine()
	if err != nil {
		t.Error(err)
	}
	if string(line) != "line1" {
		t.Error("first line mismatch=")
	}
	nft := newFileForTest(t)
	newReader, err := NewLineReaderAtPosition(nft.Reader, position, false)
	if err != nil {
		t.Error(err)
	}
	assertReadLineFindsLines(t, newReader, []string{"line2", "line3"})
}
