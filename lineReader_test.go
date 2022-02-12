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

// Ensures that a reader hits EOF when calling scan.
// It also ensures that Scan() doesn't return any error.
func (lr *LineReader) assertScanReachesEOF(t *testing.T) {
	line, _, err := lr.ReadLine()
	if err != io.EOF {
		t.Error(err)
	}
	if line != nil {
		t.Errorf("reader found an unexpected line: '%s'", line)
	}
}

// Ensures that calling Scan() once will find the expected line
// Note that lines are typically []byte, but we're using strings here for convenience.
func (lr *LineReader) assertScanFindsLine(t *testing.T, expected string) {
	line, _, err := lr.ReadLine()
	if err != nil {
		t.Error(err)
	}
	if string(line) != expected {
		t.Errorf("Expected to find line:\n%s\nBut found\n%s", expected, line)
	}
}

// Ensures that the reader would find the expected lines.
// It doesn't continue scanning after the last matched line.
func (lr *LineReader) assertScanFindsLines(t *testing.T, expected []string) {
	for _, line := range expected {
		lr.assertScanFindsLine(t, line)
	}
}

// Test that calling Scan() on an empty file will just hit EOF
func TestEmptyFileReturnsEOF(t *testing.T) {
	reader := NewLineReader(iotest.ErrReader(io.EOF))
	reader.assertScanReachesEOF(t)
}

// Test that calling Scan() on a file with a single and terminated line will find it.
func TestSingleLineIsFound(t *testing.T) {
	reader := NewLineReader(strings.NewReader("Hello World\n"))
	reader.assertScanFindsLine(t, "Hello World")
}

// Test that a file holding two terminated lines will be found by calling ReadLine() repeatedly.
func TestTwoLinesAreFound(t *testing.T) {
	tf := newFileForTest(t)
	reader := NewLineReader(tf.Reader)
	tf.Append("Hello world\nAnother\n")
	reader.assertScanFindsLines(t, []string{"Hello world", "Another"})
}

// Test that an unterminated line at the end of a file will not be returned by ReadLine,
// and ReadLastLine will return it
func TestUnterminatedLineAtEOFNotReadImplicitly(t *testing.T) {
	tf := newFileForTest(t)
	reader := NewLineReader(tf.Reader)
	tf.Append("Hello world\nUnterminated")
	reader.assertScanFindsLines(t, []string{"Hello world"})
	lastLine, err := reader.ReadLastLine()
	if err != nil {
		t.Error(err)
	}
	if string(lastLine) != "Unterminated" {
		t.Errorf("Unterminated line at EOF read incorrectly. got %s", lastLine)
	}
}

// Test that we don't remove a trailing \r from an unterminated line read by ReadLastLine()
func TestCRNotRemovedFromUnterminatedLine(t *testing.T) {
	tf := newFileForTest(t)
	reader := NewLineReader(tf.Reader)
	reader.assertScanFindsLines(t, nil)
	tf.Append("Unterminated\r")
	lastLine, err := reader.ReadLastLine()
	if err != nil {
		t.Error(err)
	}
	if string(lastLine) != "Unterminated\r" {
		t.Errorf("Unterminated line at EOF read incorrectly. Got %s", lastLine)
	}
}

// A reader han hit eof, but if the file gets written by a third party, it will find those
//appended contents on the next call to Scan()
func TestCanReadAppendsAfterReachingEOF(t *testing.T) {
	ft := newFileForTest(t)
	reader := NewLineReader(ft.Reader)

	ft.Append("Hello ")
	reader.assertScanReachesEOF(t)

	ft.Append("World\n")
	reader.assertScanFindsLine(t, "Hello World")
}

// A reader will not scan a lien unless it's finished by a newline, even at EOF.
// This is relevant because a half written line at the end of the file is ambiguous:
// It might be the last (but unterminated!) full line of a file, or it's a partially written line.
// Typically, the reader would need more context in order to determine this.
// For example, a multiple file reader would consider a file closed (and immutable) after the
// next one is found, hence disambiguating that last fragment as a whole line.
func TestWaitsForNewlineAtEOF(t *testing.T) {
	ft := newFileForTest(t)
	reader := NewLineReader(ft.Reader)

	ft.Append("Hello World\nHey!")
	reader.assertScanFindsLine(t, "Hello World")
	reader.assertScanReachesEOF(t)
	ft.Append("\n")
	reader.assertScanFindsLine(t, "Hey!")
}

// helper function to create a string full of zeroes.
func makeZeroesString(length uint) string {
	return fmt.Sprintf("%*d", length, 0)
}

// Test that the if the underlying scanned file is served by a buffered reader,
// LineReader can find lines larger than the buffer size.
func TestLineCanBeBiggerThanBufferSize(t *testing.T) {
	contents := makeZeroesString(1000)
	ft := newFileForTest(t)
	ft.Append(contents + "\n")
	reader := NewLineReader(nil)
	reader.reader = bufio.NewReaderSize(ft.Reader, 5)
	reader.assertScanFindsLine(t, contents)
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
	newReader, err := NewLineReaderAtPosition(nft.Reader, position)
	if err != nil {
		t.Error(err)
	}
	newReader.assertScanFindsLines(t, []string{"line2", "line3"})
}
