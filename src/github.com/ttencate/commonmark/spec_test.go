package commonmark

import (
	"bufio"
	"bytes"
	"fmt"
	"go/build"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

type example struct {
	section string
	number  int
	input   []byte
	output  []byte
}

type result struct {
	run    int
	failed int
}

func TestSpec(t *testing.T) {
	specFile, err := openSpecFile()
	if err != nil {
		t.Fatalf("error loading spec.txt: %s", err)
	}

	examples := make(chan example)
	go readExamples(specFile, examples)

	var res result
	var sections []string
	sectionResults := make(map[string]*result)
	for ex := range examples {
		var failed bool
		actualOutput, err := ToHTMLBytes(ex.input)
		if err != nil {
			failed = true
			t.Errorf("error in example %d: %s\ninput:\n%s", ex.number, err, ex.input)
		} else if !bytes.Equal(actualOutput, ex.output) {
			failed = true
			t.Errorf("incorrect output in section \"%s\" example %d\ninput:\n%s\nexpected output:\n%s\nactual output:\n%s",
				ex.section, ex.number, ex.input, ex.output, actualOutput)
		}

		if sectionResults[ex.section] == nil {
			sections = append(sections, ex.section)
			sectionResults[ex.section] = &result{}
		}
		res.run++
		sectionResults[ex.section].run++
		if failed {
			res.failed++
			sectionResults[ex.section].failed++
		}
	}
	output := "spec test complete\n"
	output += fmt.Sprintf("%-28s   CNT  PASS  FAIL\n", "")
	for _, section := range sections {
		res := sectionResults[section]
		output += fmt.Sprintf("%-28s   %3d   %3d   %3d\n", section, res.run, res.run-res.failed, res.failed)
	}
	output += fmt.Sprintf("%-28s   %3d   %3d   %3d\n", "TOTAL", res.run, res.run-res.failed, res.failed)
	t.Log(output)
}

func openSpecFile() (*os.File, error) {
	pkg, err := build.Import("github.com/ttencate/commonmark", "", build.FindOnly)
	if err != nil {
		return nil, err
	}

	filename := filepath.Join(pkg.Root, "spec.txt")

	return os.Open(filename)
}

var headerRegexp = regexp.MustCompile("^#{1,6} (.*)$")

func readExamples(reader io.Reader, examples chan<- example) {
	scanner := bufio.NewScanner(reader)

	// Loosely based on https://github.com/jgm/CommonMark/blob/master/runtests.pl
	// Example syntax is:
	//
	// .
	// markdown input
	// .
	// html output
	// .
	var stage int
	var input, output []byte
	nextNumber := 1
	var section string
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 1 && line[0] == '.' {
			switch stage {
			case 0:
				stage = 1
			case 1:
				stage = 2
			case 2:
				examples <- example{section, nextNumber, replaceMagicChars(input), replaceMagicChars(output)}
				nextNumber++
				input = nil
				output = nil
				stage = 0
			}
		} else {
			switch stage {
			case 0:
				if m := headerRegexp.FindSubmatch(line); m != nil {
					section = string(m[1])
				}
			case 1:
				input = append(input, line...)
				input = append(input, '\n')
			case 2:
				output = append(output, line...)
				output = append(output, '\n')
			}
		}
	}

	if err := scanner.Err(); err != nil {
		log.Panicf("error while reading spec file: %s", err)
	}

	close(examples)
}

func replaceMagicChars(text []byte) []byte {
	text = bytes.Replace(text, []byte("→"), []byte("\t"), -1)
	text = bytes.Replace(text, []byte("␣"), []byte(" "), -1)
	return text
}
