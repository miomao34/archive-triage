package main

import (
	"fmt"
	"html"
	linkreader "miomao34/archive-triage/link_reader"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/log"
)

type ExportFormatType int

const (
	MarkdownExportFormat ExportFormatType = iota
	BookmarkExportFormat ExportFormatType = iota
)

type LinkWithTags struct {
	link linkreader.Linker
	tags []string
}

func saveOneLinkToMarkdown(dir string, params LinkWithTags) error {
	filename := filepath.Join(dir, string(params.link.GetName())+".md")
	os.Create(filename)
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o666)
	if err != nil {
		return err
	}
	defer file.Close()

	var b strings.Builder

	b.WriteString("---\ntags:\n")
	for _, tag := range params.tags {
		b.WriteString("  - ")
		b.WriteString(tag)
		b.WriteRune('\n')
	}
	b.WriteString("---\n")

	b.WriteString(string(params.link.GetHREF()))
	b.WriteRune('\n')

	file.WriteString(b.String())
	file.Sync()

	return nil
}

func saveLinkWorker(goInput <-chan LinkWithTags, dir string) {
	for input := range goInput {
		err := saveOneLinkToMarkdown(dir, input)
		if err != nil {
			log.Error("failed to write link to file",
				"err", err,
				"link", input.link.GetHREF(),
			)
		}
	}
}

func saveLinksToMarkdown(m *model, dir string) error {
	err := os.MkdirAll(dir, 0o755)
	if err != nil {
		return err
	}
	linkIDs, savedLinks, err := m.conn.GetAllSavedLinks()
	if err != nil {
		return err
	}

	numberOfWorkers := 4
	wg := &sync.WaitGroup{}
	goInput := make(chan LinkWithTags)
	for i := 0; i <= numberOfWorkers; i++ {
		wg.Go(func() { saveLinkWorker(goInput, dir) })
	}

	for seqID, link := range savedLinks {
		tags, err := m.conn.GetLinkTags(linkIDs[seqID])
		if err != nil {
			return err
		}

		goInput <- LinkWithTags{link, tags}
	}
	close(goInput)
	wg.Wait()

	return nil
}

func saveLinksToBookmarkFile(m *model, filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		log.Error("failed to create file", "err", err)
		return err
	}
	defer f.Close()
	f.WriteString(`<!DOCTYPE NETSCAPE-Bookmark-file-1>
<!-- This is an automatically generated file.
     It will be read and overwritten.
     DO NOT EDIT! -->
<META HTTP-EQUIV="Content-Type" CONTENT="text/html; charset=UTF-8">
<TITLE>Bookmarks</TITLE>
<H1>Bookmarks</H1>
<DL><p>
`)
	fmt.Fprintf(f, `    <DT><H3 ADD_DATE="%v" LAST_MODIFIED="%v" PERSONAL_TOOLBAR_FOLDER="true">archive-triage export</H3>
    <DL><p>
`, 1, time.Now().Unix())

	_, savedLinks, err := m.conn.GetAllSavedLinks()
	if err != nil {
		return err
	}

	for id, link := range savedLinks {
		fmt.Fprintf(f, "        <DT><A HREF=\"%v\" ADD_DATE=\"%v\" ICON=\"\">%v</A>\n",
			string(link.GetHREF()),
			id+1,
			html.EscapeString(string(link.GetName())))
	}
	f.WriteString(`    </DL><p>
</DL><p>`)

	return nil
}
