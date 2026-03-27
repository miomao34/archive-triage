// this package is responsible for parsing link tags
// and returning Link objects is a generator-esque fashion
// possible improvements:
// - incremental/chunked file reads for very large files processing
// -

package linkreader

import (
	"fmt"
	"io"
	"os"
	"regexp"
)

type LinkFileFormatType int

const (
	ExtensionExportFormat        LinkFileFormatType = iota
	BookmarkExportFormat         LinkFileFormatType = iota
	FirefoxShareTabsExportFormat LinkFileFormatType = iota
)

type Link struct {
	name []byte
	href []byte
}

func (l Link) GetName() []byte {
	return l.name
}

func (l Link) GetHREF() []byte {
	return l.href
}

func (l *Link) SetName(name []byte) {
	l.name = name
}

func (l *Link) SetHREF(href []byte) {
	l.href = href
}

type Linker interface {
	GetName() []byte
	GetHREF() []byte
}

type LinkGenerator struct {
	ReturnChannel chan Linker

	fileContents []byte
	readOffset   int

	lastResult map[string][]byte

	formatType LinkFileFormatType
	pattern    *regexp.Regexp
}

func (lg *LinkGenerator) ReadBookmarksFile(filename string, format LinkFileFormatType) error {
	var err error

	fd, err := os.Open(filename)
	if err != nil {
		// todo: logging hehe
		return err
	}
	defer fd.Close()

	lg.ReturnChannel = make(chan Linker, 2)
	lg.formatType = format
	switch format {
	case BookmarkExportFormat:
		lg.pattern, err = regexp.Compile(`(?ismU)<\s*a\s+.*href\s*=\s*\"(?<href>.*)\".*>(?<name>.*)<\s*/\s*a\s*>`)
		if err != nil {
			os.Exit(-1)
		}
	case ExtensionExportFormat:
		// bg.pattern, err = regexp.Compile(`(?ismU)^(?<name>.*?)$\n^(?<href>http[s]*://.*?)$`)
		lg.pattern, err = regexp.Compile(`(?ismU)^(?<name>[^\n]*)$\n^(?<href>http[s]*://[^\n]*)\n$`)
		// lg.pattern, err = regexp.Compile(`(?ismU)^(?<name>.*)\n$^(?<href>http[s]*://[^\n]*)\n$`)
		if err != nil {
			os.Exit(-1)
		}
	case FirefoxShareTabsExportFormat:
		// bg.pattern, err = regexp.Compile(`(?ismU)^(?<name>.*?)$\n^(?<href>http[s]*://.*?)$`)
		lg.pattern, err = regexp.Compile(`(?ismU)^(?<href>http[s]*://[^\n]*)\n\n$`)
		// lg.pattern, err = regexp.Compile(`(?ismU)^(?<name>.*)\n$^(?<href>http[s]*://[^\n]*)\n$`)
		if err != nil {
			os.Exit(-1)
		}
	}

	lg.fileContents, err = io.ReadAll(fd)
	if err != nil {
		return err
	}

	lg.readOffset = 0

	lg.lastResult = make(map[string][]byte, 1)

	return nil
}

// structure: <a href="[link]" ... >[title]<a>
// params can be uppercase
func (lg *LinkGenerator) GetNextLink() (Linker, error) {

	if lg.readOffset >= len(lg.fileContents) {
		// you have reached the end.
		return nil, nil
	}

	searchSlice := lg.fileContents[lg.readOffset:]

	match := lg.pattern.FindSubmatchIndex(searchSlice)
	if match == nil {
		return nil, nil
	}
	for i, name := range lg.pattern.SubexpNames() {
		if i == 0 {
			continue
		}
		lg.lastResult[name] = searchSlice[match[i*2]:match[i*2+1]]
	}

	lg.readOffset += match[1] + 1

	var name, href []byte
	name, _ = lg.lastResult["name"]
	href, _ = lg.lastResult["href"]
	return Link{name: name, href: href}, nil
}

func (lg *LinkGenerator) Run() error {
	go func() {
		// var link Linker
		// var err error
		for link, err := lg.GetNextLink(); link != nil; link, err = lg.GetNextLink() {
			if err != nil {
				// hehe
				fmt.Println("oh lawd")
			}
			lg.ReturnChannel <- link
		}

		close(lg.ReturnChannel)
	}()

	return nil
}
