package commonmark

import (
  "bufio"
  "go/build"
  "path/filepath"
  "io"
  "os"
  "testing"
)

type example struct {
  input string
  output string
}

func TestSpec(t *testing.T) {
  specFile, err := openSpecFile()
  if err != nil {
    t.Fatalf("Failed to load spec.txt: %s", err)
  }

  examples := make(chan example)
  go readExamples(specFile, examples)

  for ex := range examples {
    actualOutput := ToHTML(ex.input)
    if actualOutput != ex.output {
      t.Errorf("Input:\n%s\nActual output:\n%s\nExpected output:\n%s",
          ex.input, actualOutput, ex.output)
    }
  }
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

  // Loosely based on https://github.com/jgm/CommonMark/blob/master/spec2md.pl
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
      switch (stage) {
      case 0:
        input = nil
        output = nil
        stage = 1
      case 1:
        stage = 2
      case 2:
        examples <- example{string(input), string(output)}
        stage = 0
      }
    } else {
      switch (stage) {
      case 1:
        input = append(input, line...)
        input = append(input, '\n')
      case 2:
        output = append(output, line...)
        output = append(output, '\n')
      }
    }
  }

  close(examples)
}
