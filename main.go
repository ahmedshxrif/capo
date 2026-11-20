package main

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

const usage = `capo: transpose chord sheets

  capo up <semitones> [file]     move every chord up (or down with a minus number)
  capo to <key> [file]           move the song into another key, for example "capo to G song.txt"
  capo fret <number> [file]      the shapes to play with a capo on that fret
  capo key [file]                guess the key of the song
  capo numbers [file]            show the chords as Nashville numbers in the song's key

Reads from the file, or from standard input when no file is given.
Add --flats or --sharps to choose how notes are spelled.`

func readSheet(path string) (string, error) {
	if path == "" || path == "-" {
		data, err := io.ReadAll(os.Stdin)
		return string(data), err
	}
	data, err := os.ReadFile(path)
	return string(data), err
}

func run(args []string, stdout io.Writer) error {
	var flats, sharps bool
	var rest []string
	for _, arg := range args {
		switch arg {
		case "--flats":
			flats = true
		case "--sharps":
			sharps = true
		default:
			rest = append(rest, arg)
		}
	}
	if len(rest) == 0 {
		fmt.Fprintln(stdout, usage)
		return nil
	}
	command := rest[0]
	arg := func(i int) string {
		if len(rest) > i {
			return rest[i]
		}
		return ""
	}

	switch command {
	case "key":
		sheet, err := readSheet(arg(1))
		if err != nil {
			return err
		}
		key, minor := DetectKey(chordsIn(sheet))
		if key == "" {
			return fmt.Errorf("no chords found")
		}
		if minor {
			fmt.Fprintf(stdout, "%s minor\n", key)
		} else {
			fmt.Fprintf(stdout, "%s major\n", key)
		}
		return nil

	case "numbers":
		sheet, err := readSheet(arg(1))
		if err != nil {
			return err
		}
		key, minor := DetectKey(chordsIn(sheet))
		if key == "" {
			return fmt.Errorf("no chords found")
		}
		fmt.Fprint(stdout, NumberSheet(sheet, key, minor))
		return nil

	case "up", "fret":
		steps, err := strconv.Atoi(arg(1))
		if err != nil {
			return fmt.Errorf("%q is not a whole number of semitones", arg(1))
		}
		if command == "fret" {
			steps = -steps // a capo raises the pitch, so the shapes you play are lower
		}
		sheet, err := readSheet(arg(2))
		if err != nil {
			return err
		}
		fmt.Fprint(stdout, TransposeSheet(sheet, steps, flats, sharps))
		return nil

	case "to":
		target := arg(1)
		if len(target) > 0 {
			target = strings.ToUpper(target[:1]) + target[1:]
		}
		targetMinor := strings.HasSuffix(target, "m")
		target = strings.TrimSuffix(target, "m")
		if _, ok := noteIndex[target]; !ok {
			return fmt.Errorf("%q is not a key. Try something like G, Bb or F#m", arg(1))
		}
		sheet, err := readSheet(arg(2))
		if err != nil {
			return err
		}
		key, minor := DetectKey(chordsIn(sheet))
		if key == "" {
			return fmt.Errorf("no chords found")
		}
		if minor != targetMinor {
			return fmt.Errorf("the song is in %s %s, so pick a %s key", key, mode(minor), mode(minor))
		}
		fmt.Fprint(stdout, TransposeSheet(sheet, semitonesBetween(key, target), flats, sharps))
		return nil
	}
	return fmt.Errorf("unknown command %q\n\n%s", command, usage)
}

func mode(minor bool) string {
	if minor {
		return "minor"
	}
	return "major"
}

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "capo:", err)
		os.Exit(1)
	}
}
