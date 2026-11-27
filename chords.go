// Package main is capo: a small tool that transposes chord sheets.
package main

import (
	"regexp"
	"strings"
)

var (
	sharpNames = []string{"C", "C#", "D", "D#", "E", "F", "F#", "G", "G#", "A", "A#", "B"}
	flatNames  = []string{"C", "Db", "D", "Eb", "E", "F", "Gb", "G", "Ab", "A", "Bb", "B"}
	noteIndex  = map[string]int{
		"C": 0, "B#": 0, "C#": 1, "Db": 1, "D": 2, "D#": 3, "Eb": 3, "E": 4, "Fb": 4, "F": 5, "E#": 5,
		"F#": 6, "Gb": 6, "G": 7, "G#": 8, "Ab": 8, "A": 9, "A#": 10, "Bb": 10, "B": 11, "Cb": 11,
	}
	// A chord is a root, an optional quality (m, maj7, sus4, dim, add9, 7b5 ...) and an optional bass note.
	chordPattern = regexp.MustCompile(`^([A-G][#b]?)((?:maj|min|dim|aug|sus|add|m|M|\+|°|ø)?[0-9]*(?:(?:sus|add|maj|min|dim|aug|b|#|\+|-)[0-9]+)*)(?:/([A-G][#b]?))?$`)
	bracketed    = regexp.MustCompile(`\[([^\]\s]+)\]`)
	// Things that sit on a chord line without being chords: bar lines, repeats, "no chord".
	// A ChordPro directive is a whole line such as {title: Harbour Road}. Only {key: G} is rewritten.
	directive    = regexp.MustCompile(`^\s*\{.*\}\s*$`)
	keyDirective = regexp.MustCompile(`(?i)^(\s*\{\s*key\s*:\s*)([^}\s]+)(.*)$`)
	filler       = regexp.MustCompile(`^(?:\||\|\||/|-|%|N\.?C\.?|x\d+|\(?x\d+\)?|\d+x)$`)
)

// Chord is one parsed chord, such as F#m7/C#.
type Chord struct {
	Root    string
	Quality string
	Bass    string
}

// ParseChord reads a single token. ok is false when the token is not a chord.
func ParseChord(token string) (chord Chord, ok bool) {
	m := chordPattern.FindStringSubmatch(token)
	if m == nil {
		return Chord{}, false
	}
	return Chord{Root: m[1], Quality: m[2], Bass: m[3]}, true
}

// IsMinor reports whether the chord is a minor chord (Am, F#m7), as opposed to major or maj7.
func (c Chord) IsMinor() bool {
	q := c.Quality
	return (strings.HasPrefix(q, "m") && !strings.HasPrefix(q, "maj")) || strings.HasPrefix(q, "min")
}

func shiftNote(note string, semitones int, flats bool) string {
	index := ((noteIndex[note]+semitones)%12 + 12) % 12
	if flats {
		return flatNames[index]
	}
	return sharpNames[index]
}

// Transpose moves the chord by a number of semitones. flats chooses Bb over A#.
func (c Chord) Transpose(semitones int, flats bool) Chord {
	out := Chord{Root: shiftNote(c.Root, semitones, flats), Quality: c.Quality}
	if c.Bass != "" {
		out.Bass = shiftNote(c.Bass, semitones, flats)
	}
	return out
}

func (c Chord) String() string {
	if c.Bass != "" {
		return c.Root + c.Quality + "/" + c.Bass
	}
	return c.Root + c.Quality
}

// isChordLine reports whether a line holds only chords (and bar lines), which is how
// most chord sheets are written: a line of chords above a line of lyrics.
func isChordLine(line string) bool {
	fields := strings.Fields(line)
	chords := 0
	for _, field := range fields {
		if filler.MatchString(field) {
			continue
		}
		if _, ok := ParseChord(field); !ok {
			return false
		}
		chords++
	}
	return chords > 0
}

// chordsIn lists every chord in a sheet, in order: chord lines and [bracketed] chords.
func chordsIn(sheet string) []Chord {
	var found []Chord
	for _, line := range strings.Split(sheet, "\n") {
		if directive.MatchString(line) {
			continue // directives hold titles and comments, not the song's chords
		}
		if isChordLine(line) {
			for _, field := range strings.Fields(line) {
				if chord, ok := ParseChord(field); ok {
					found = append(found, chord)
				}
			}
			continue
		}
		for _, m := range bracketed.FindAllStringSubmatch(line, -1) {
			if chord, ok := ParseChord(m[1]); ok {
				found = append(found, chord)
			}
		}
	}
	return found
}

// prefersFlats decides how to spell the result. It looks at the key the song lands in and
// uses flats for the keys musicians write with flats (F, Bb, Eb, Ab, Db), sharps otherwise.
// When no key can be found it follows whatever the sheet already uses.
func prefersFlats(chords []Chord, semitones int) bool {
	key, minor := DetectKey(chords)
	if key == "" {
		flats, sharps := 0, 0
		for _, chord := range chords {
			if strings.HasSuffix(chord.Root, "b") {
				flats++
			} else if strings.HasSuffix(chord.Root, "#") {
				sharps++
			}
		}
		return flats > sharps
	}
	index := ((noteIndex[key]+semitones)%12 + 12) % 12
	if minor {
		index = (index + 3) % 12 // spell a minor key like its relative major
	}
	switch index {
	case 5, 10, 3, 8, 1: // F, Bb, Eb, Ab, Db
		return true
	}
	return false
}

// transposeChordLine replaces each chord and keeps the ones after it in the same
// column where it can, so chords stay above the right syllables.
func transposeChordLine(line string, semitones int, flats bool) string {
	return rewriteChordLine(line, func(c Chord) string { return c.Transpose(semitones, flats).String() })
}

// rewriteChordLine swaps every chord on a chord line for whatever rewrite returns, and keeps
// the columns of the chords that follow where it can.
func rewriteChordLine(line string, rewrite func(Chord) string) string {
	var out strings.Builder
	debt := 0 // columns the output is ahead of (+) or behind (-) the original
	i := 0
	for i < len(line) {
		if line[i] == ' ' || line[i] == '\t' {
			start := i
			for i < len(line) && (line[i] == ' ' || line[i] == '\t') {
				i++
			}
			gap := line[start:i]
			if strings.Contains(gap, "\t") {
				out.WriteString(gap)
				debt = 0
				continue
			}
			width := len(gap) - debt
			if width < 1 && i < len(line) {
				debt = 1 - (len(gap) - debt)
				width = 1 // never glue two chords together
			} else {
				debt = 0
			}
			out.WriteString(strings.Repeat(" ", width))
			continue
		}
		start := i
		for i < len(line) && line[i] != ' ' && line[i] != '\t' {
			i++
		}
		token := line[start:i]
		if chord, ok := ParseChord(token); ok {
			moved := rewrite(chord)
			out.WriteString(moved)
			debt += len(moved) - len(token)
		} else {
			out.WriteString(token)
		}
	}
	return out.String()
}

// transposeKeyDirective moves the key in a {key: G} line and leaves every other directive alone.
func transposeKeyDirective(line string, semitones int, flats bool) string {
	m := keyDirective.FindStringSubmatch(line)
	if m == nil {
		return line
	}
	chord, ok := ParseChord(m[2])
	if !ok || chord.Bass != "" {
		return line
	}
	return m[1] + chord.Transpose(semitones, flats).String() + m[3]
}

// TransposeSheet moves every chord in a sheet by a number of semitones and leaves the lyrics alone.
func TransposeSheet(sheet string, semitones int, forceFlats, forceSharps bool) string {
	flats := prefersFlats(chordsIn(sheet), semitones)
	if forceFlats {
		flats = true
	}
	if forceSharps {
		flats = false
	}
	lines := strings.Split(sheet, "\n")
	for i, line := range lines {
		if directive.MatchString(line) {
			lines[i] = transposeKeyDirective(line, semitones, flats)
			continue
		}
		if isChordLine(line) {
			lines[i] = transposeChordLine(line, semitones, flats)
			continue
		}
		lines[i] = bracketed.ReplaceAllStringFunc(line, func(match string) string {
			if chord, ok := ParseChord(match[1 : len(match)-1]); ok {
				return "[" + chord.Transpose(semitones, flats).String() + "]"
			}
			return match
		})
	}
	return strings.Join(lines, "\n")
}

// DetectKey guesses the key of a song from its chords. It scores every major key by how many
// chords belong to it, with extra weight on the chord the song starts and ends on.
// minor is true when the song centres on the relative minor (Am for C major).
func DetectKey(chords []Chord) (key string, minor bool) {
	if len(chords) == 0 {
		return "", false
	}
	// Scale steps of the chords that belong to a major key, and whether each is minor.
	diatonic := map[int]bool{0: false, 2: true, 4: true, 5: false, 7: false, 9: true}
	best, bestScore := 0, -1.0
	for tonic := 0; tonic < 12; tonic++ {
		score := 0.0
		for i, chord := range chords {
			step := ((noteIndex[chord.Root]-tonic)%12 + 12) % 12
			wantMinor, in := diatonic[step]
			if !in || wantMinor != chord.IsMinor() {
				continue
			}
			weight := 1.0
			if step == 0 || step == 9 {
				weight = 1.5 // the home chord, or its relative minor
			}
			if i == 0 || i == len(chords)-1 {
				weight += 1.5
			}
			score += weight
		}
		if score > bestScore {
			best, bestScore = tonic, score
		}
	}
	first, last := chords[0], chords[len(chords)-1]
	relative := (best + 9) % 12
	if first.IsMinor() && last.IsMinor() && noteIndex[first.Root] == relative {
		return spellKey(relative), true
	}
	return spellKey(best), false
}

func spellKey(index int) string {
	switch index {
	case 1, 3, 8, 10: // Db, Eb, Ab, Bb are the usual spellings
		return flatNames[index]
	}
	return sharpNames[index]
}

// semitonesBetween is the shortest way up or down from one note to another.
func semitonesBetween(from, to string) int {
	distance := ((noteIndex[to]-noteIndex[from])%12 + 12) % 12
	if distance > 6 {
		distance -= 12
	}
	return distance
}

var degreeNames = []string{"1", "b2", "2", "b3", "3", "4", "b5", "5", "b6", "6", "b7", "7"}

// degree is the Nashville number of a note in the key whose tonic is tonic.
func degree(note, tonic string) string {
	return degreeNames[((noteIndex[note]-noteIndex[tonic])%12+12)%12]
}

// Nashville writes the chord as a number in the key, so Em in G is 6m and D/F# is 5/7.
func (c Chord) Nashville(tonic string) string {
	out := degree(c.Root, tonic) + c.Quality
	if c.Bass != "" {
		out += "/" + degree(c.Bass, tonic)
	}
	return out
}

// NumberSheet replaces every chord with its Nashville number. A minor key counts from its
// relative major, the way DetectKey sees it, so Am in A minor is 6m.
func NumberSheet(sheet, key string, minor bool) string {
	tonic := key
	if minor {
		tonic = flatNames[(noteIndex[key]+3)%12]
	}
	rewrite := func(c Chord) string { return c.Nashville(tonic) }
	lines := strings.Split(sheet, "\n")
	for i, line := range lines {
		if directive.MatchString(line) {
			continue // the key directive is a plain name, so it stays as written
		}
		if isChordLine(line) {
			lines[i] = rewriteChordLine(line, rewrite)
			continue
		}
		lines[i] = bracketed.ReplaceAllStringFunc(line, func(match string) string {
			if chord, ok := ParseChord(match[1 : len(match)-1]); ok {
				return "[" + rewrite(chord) + "]"
			}
			return match
		})
	}
	return strings.Join(lines, "\n")
}
