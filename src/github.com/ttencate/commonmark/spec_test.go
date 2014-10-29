package commonmark

import (
	"bufio"
	"bytes"
	"go/build"
	"io"
	"log"
	"os"
	"path/filepath"
	"testing"
)

type example struct {
	input  []byte
	output []byte
}

func TestSpec(t *testing.T) {
	specFile, err := openSpecFile()
	if err != nil {
		t.Fatalf("error loading spec.txt: %s", err)
	}

	examples := make(chan example)
	go readExamples(specFile, examples)

	var count, failures int
	for ex := range examples {
		count++
		actualOutput, err := ToHTMLBytes(ex.input)
		if err != nil {
			failures++
			t.Errorf("error: %s\ninput:\n%s", err, ex.input)
			continue
		}
		if !bytes.Equal(actualOutput, ex.output) {
			failures++
			t.Errorf("incorrect output\ninput:\n%s\nexpected output:\n%s\nactual output:\n%s",
				ex.input, ex.output, actualOutput)
		}
	}
	t.Logf("spec test complete\ntests run: %3d\nsuccesses: %3d\nfailures:  %3d", count, count-failures, failures)
}

func openSpecFile() (*os.File, error) {
	pkg, err := build.Import("github.com/ttencate/commonmark", "", build.FindOnly)
	if err != nil {
		return nil, err
	}

	filename := filepath.Join(pkg.Root, "spec.txt")

	return os.Open(filename)
}

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
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 1 && line[0] == '.' {
			switch stage {
			case 0:
				stage = 1
			case 1:
				stage = 2
			case 2:
				examples <- example{replaceMagicChars(input), replaceMagicChars(output)}
				input = nil
				output = nil
				stage = 0
			}
		} else {
			switch stage {
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
