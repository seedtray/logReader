package main

import (
	"flag"
	"fmt"
	"github.com/seedtray/logReader"
	"github.com/spf13/afero"
	"io"
	"log"
)

// Prints a file's lines and watches for further appends. Similar to tail -f.
// Each line is printed prefixed by a number which can be used as a starting point on a later call.
func main() {

	var position = flag.Int64("resume", 0, "Resume from position. By default it starts at the beginning.")
	flag.Parse()
	filename := flag.Arg(0)

	watcher := logReader.NewOsPollingFileWatcher(filename)
	fileUpdates, stop := watcher.Start()
	defer stop()

	fs := afero.NewOsFs()
	file, err := fs.Open(filename)
	if err != nil {
		log.Fatalln(err)
	}
	lineReader, err := logReader.NewLineReaderAtPosition(file, *position, false)
	if err != nil {
		log.Fatalln(err)
	}

	for {
		line, position, err := lineReader.ReadLine()
		if err == nil {
			fmt.Printf("%10d: %s\n", position, string(line))
		} else if err == io.EOF {
			_, ok := <-fileUpdates
			if !ok {
				log.Fatalln(watcher.Err())
			}
		} else {
			log.Fatalln(err)
		}
	}
}
