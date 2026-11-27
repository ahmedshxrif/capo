package main

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestParseChord(t *testing.T) {
	good := map[string]Chord{
		"C":      {Root: "C"},
		"F#m7":   {Root: "F#", Quality: "m7"},
		"Bbmaj7": {Root: "Bb", Quality: "maj7"},
		"D/F#":   {Root: "D", Bass: "F#"},
		"Asus4":  {Root: "A", Quality: "sus4"},
		"G7b5":   {Root: "G", Quality: "7b5"},
		"Cadd9":  {Root: "C", Quality: "add9"},
		"Edim":   {Root: "E", Quality: "dim"},
	}
	for text, want := range good {
		got, ok := ParseChord(text)
		if !ok || got != want {
			t.Errorf("ParseChord(%q) = %+v, %v; want %+v", text, got, ok, want)
		}
	}
	for _, text := range []string{"Hello", "A-ha", "Am I", "H", "Cat", "Dance", "", "Go"} {
		if _, ok := ParseChord(text); ok {
			t.Errorf("ParseChord(%q) should not be a chord", text)
		}
	}
}

func TestTransposeChord(t *testing.T) {
	cases := []struct {
		in    string
		steps int
		flats bool
		want  string
	}{
		{"C", 2, false, "D"},
		{"Am7", 3, false, "Cm7"},
		{"D/F#", 2, false, "E/G#"},
		{"B", 1, false, "C"},
		{"C", -1, false, "B"},
		{"A", 1, true, "Bb"},
		{"A", 1, false, "A#"},
		{"Gsus4", 12, false, "Gsus4"},
		{"Eb", -3, false, "C"},
	}
	for _, c := range cases {
		chord, _ := ParseChord(c.in)
		if got := chord.Transpose(c.steps, c.flats).String(); got != c.want {
			t.Errorf("%s %+d = %s, want %s", c.in, c.steps, got, c.want)
		}
	}
}

func TestChordLinesAreTransposedAndLyricsAreNot(t *testing.T) {
	sheet := "G       D\nAm I going home\nEm      C"
	want := "A       E\nAm I going home\nF#m     D"
	if got := TransposeSheet(sheet, 2, false, false); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestChordsStayInTheirColumns(t *testing.T) {
	// G to G# is one character wider, C# to D one narrower: the next chord must not drift.
	got := TransposeSheet("G    C#   D", 1, false, true)
	if got != "G#   D    D#" {
		t.Errorf("got %q", got)
	}
}

func TestChordsNeverRunTogether(t *testing.T) {
	got := TransposeSheet("G C", 1, false, true)
	if got != "G# C#" {
		t.Errorf("got %q", got)
	}
}

func TestBracketedChords(t *testing.T) {
	got := TransposeSheet("[Am]Say it [C]slow [not a chord]", 2, false, false)
	if got != "[Bm]Say it [D]slow [not a chord]" {
		t.Errorf("got %q", got)
	}
}

func TestBarLinesAndRepeatsStayPut(t *testing.T) {
	got := TransposeSheet("| G | C | x2", 2, false, false)
	if got != "| A | D | x2" {
		t.Errorf("got %q", got)
	}
}

func TestSpellingFollowsTheKey(t *testing.T) {
	sheet := "G  C  D  G"
	if got := TransposeSheet(sheet, 3, false, false); got != "Bb Eb F  Bb" {
		t.Errorf("up 3 to Bb: got %q", got)
	}
	if got := TransposeSheet(sheet, 2, false, false); got != "A  D  E  A" {
		t.Errorf("up 2 to A: got %q", got)
	}
	if got := TransposeSheet(sheet, 3, false, true); got != "A# D# F  A#" {
		t.Errorf("forced sharps: got %q", got)
	}
}

func TestDetectKey(t *testing.T) {
	cases := []struct {
		chords string
		key    string
		minor  bool
	}{
		{"G D Em C G", "G", false},
		{"C G Am F C", "C", false},
		{"Am F C G Am", "A", true},
		{"D A Bm G D", "D", false},
		{"Bb Eb F Bb", "Bb", false},
		{"Em C G D Em", "E", true},
	}
	for _, c := range cases {
		key, minor := DetectKey(chordsIn(c.chords))
		if key != c.key || minor != c.minor {
			t.Errorf("%s: got %s minor=%v, want %s minor=%v", c.chords, key, minor, c.key, c.minor)
		}
	}
	if key, _ := DetectKey(nil); key != "" {
		t.Errorf("no chords should give no key, got %q", key)
	}
}

func TestTransposingTwelveSemitonesChangesNothing(t *testing.T) {
	data, err := os.ReadFile("examples/song.txt")
	if err != nil {
		t.Fatal(err)
	}
	sheet := string(data)
	if got := TransposeSheet(sheet, 12, false, false); got != sheet {
		t.Errorf("a full octave should give the same sheet back:\n%s", got)
	}
	there := TransposeSheet(sheet, 5, false, false)
	if back := TransposeSheet(there, -5, false, false); back != sheet {
		t.Errorf("up 5 then down 5 should give the same sheet back:\n%s", back)
	}
}

func TestCommands(t *testing.T) {
	run1 := func(args ...string) (string, error) {
		var out bytes.Buffer
		err := run(args, &out)
		return out.String(), err
	}
	if out, err := run1("key", "examples/song.txt"); err != nil || strings.TrimSpace(out) != "G major" {
		t.Errorf("key: %q %v", out, err)
	}
	if out, err := run1("to", "Bb", "examples/song.txt"); err != nil || !strings.HasPrefix(out, "Bb          F") {
		t.Errorf("to Bb: %q %v", out, err)
	}
	if out, err := run1("fret", "3", "examples/song.txt"); err != nil || !strings.HasPrefix(out, "E           B") {
		t.Errorf("fret 3: %q %v", out, err)
	}
	if _, err := run1("to", "H", "examples/song.txt"); err == nil {
		t.Error("an unknown key should be an error")
	}
	if _, err := run1("to", "Am", "examples/song.txt"); err == nil {
		t.Error("a major song cannot be moved to a minor key")
	}
	if _, err := run1("up", "two", "examples/song.txt"); err == nil {
		t.Error("semitones must be a number")
	}
	if _, err := run1("dance"); err == nil {
		t.Error("an unknown command should be an error")
	}
}

func TestNashvilleNumbers(t *testing.T) {
	cases := []struct {
		chord, tonic, want string
	}{
		{"G", "G", "1"},
		{"C", "G", "4"},
		{"D", "G", "5"},
		{"Em", "G", "6m"},
		{"Am7", "G", "2m7"},
		{"D7", "G", "57"},
		{"D/F#", "G", "5/7"},
		{"F", "G", "b7"},
		{"Bb", "G", "b3"},
		{"Fmaj7", "C", "4maj7"},
	}
	for _, c := range cases {
		chord, _ := ParseChord(c.chord)
		if got := chord.Nashville(c.tonic); got != c.want {
			t.Errorf("%s in %s = %s, want %s", c.chord, c.tonic, got, c.want)
		}
	}
}

func TestNumberSheetMajorKey(t *testing.T) {
	got := NumberSheet("G       D\nAm I going home\nEm      C", "G", false)
	want := "1       5\nAm I going home\n6m      4"
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestNumberSheetMinorKeyCountsFromRelativeMajor(t *testing.T) {
	got := NumberSheet("Am F C G", "A", true)
	if got != "6m 4 1 5" {
		t.Errorf("got %q", got)
	}
}

func TestNumberSheetKeepsColumnsAndBrackets(t *testing.T) {
	// Bb in G is b3, one character wider, so the gap after it gives the character back.
	if got := NumberSheet("Bb   D", "G", false); got != "b3   5" {
		t.Errorf("got %q", got)
	}
	if got := NumberSheet("[Am]Say it [D/F#]slow [not a chord]", "G", false); got != "[2m]Say it [5/7]slow [not a chord]" {
		t.Errorf("got %q", got)
	}
}

func TestNumbersCommand(t *testing.T) {
	var out bytes.Buffer
	if err := run([]string{"numbers", "examples/song.txt"}, &out); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(out.String(), "1           5\n") || !strings.Contains(out.String(), "[5/7]where to go") {
		t.Errorf("got:\n%s", out.String())
	}
	if err := run([]string{"numbers", "no-such-file.txt"}, &out); err == nil {
		t.Error("a missing file should be an error")
	}
}

func TestChordProDirectivesPassThrough(t *testing.T) {
	sheet, err := os.ReadFile("examples/song.cho")
	if err != nil {
		t.Fatal(err)
	}
	got := TransposeSheet(string(sheet), 3, false, false)
	want := "{title: Harbour Road}\n{artist: Ahmed Farahat}\n{key: Bb}\n{comment: Play [Am] softly}\n\n" +
		"[Bb]Morning on the [F]harbour road\n[Gm]Coffee and a [Eb]half-read code\n"
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestKeyDirectiveVariants(t *testing.T) {
	cases := map[string]string{
		"{key: Em}":       "{key: Gm}",
		"{ KEY : F# }":    "{ KEY : A }",
		"{key: nonsense}": "{key: nonsense}",
		"{subtitle: G}":   "{subtitle: G}",
	}
	for in, want := range cases {
		if got := TransposeSheet(in, 3, false, false); got != want {
			t.Errorf("%q: got %q, want %q", in, got, want)
		}
	}
}

func TestDirectivesDoNotAffectKeyOrNumbers(t *testing.T) {
	sheet := "{comment: [F] and [Bb] here}\n[G]one [D]two [C]three\n{key: G}"
	if key, _ := DetectKey(chordsIn(sheet)); key != "G" {
		t.Errorf("key = %q, want G", key)
	}
	want := "{comment: [F] and [Bb] here}\n[1]one [5]two [4]three\n{key: G}"
	if got := NumberSheet(sheet, "G", false); got != want {
		t.Errorf("got:\n%s", got)
	}
}

func TestToCommandUpdatesKeyDirective(t *testing.T) {
	var out bytes.Buffer
	if err := run([]string{"to", "A", "examples/song.cho"}, &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "{key: A}") || !strings.Contains(out.String(), "{title: Harbour Road}") {
		t.Errorf("got:\n%s", out.String())
	}
}
