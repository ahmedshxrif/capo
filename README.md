# capo

[![tests](https://github.com/ahmedshxrif/capo/actions/workflows/ci.yml/badge.svg)](https://github.com/ahmedshxrif/capo/actions/workflows/ci.yml)
![go](https://img.shields.io/badge/go-1.22%2B-00add8)
![dependencies](https://img.shields.io/badge/dependencies-none-success)
![license](https://img.shields.io/badge/license-MIT-blue)

**Transpose a chord sheet from the terminal.** The song is in G and the singer wants Bb. capo moves every chord, leaves the lyrics alone, and keeps each chord above the syllable it belongs to.

```
$ cat song.txt
G           D
Morning on the harbour road
Em          C
Coffee and a half-read code

$ capo to Bb song.txt
Bb          F
Morning on the harbour road
Gm          Eb
Coffee and a half-read code
```

## Install

```bash
go install github.com/ahmedshxrif/capo@latest
```

One small binary, no dependencies.

## Commands

| Command | What it does |
|---|---|
| `capo up 2 song.txt` | Move every chord up two semitones. Use a minus number to go down |
| `capo to Bb song.txt` | Work out the song's key and move it to Bb |
| `capo fret 3 song.txt` | Show the shapes to play with a capo on the third fret |
| `capo key song.txt` | Guess the key |
| `capo numbers song.txt` | Show the chords as Nashville numbers in the song's key |

Leave the file out to read from standard input, so it works in a pipe:

```bash
pbpaste | capo to D | pbcopy
```

Add `--flats` or `--sharps` if you want to choose the spelling yourself.

Nashville numbers write each chord as its place in the key. In G, `G C D Em` becomes `1 4 5 6m`, and `D/F#` becomes `5/7`. A minor key counts from its relative major, so `Am F C G` in A minor becomes `6m 4 1 5`.

## What it gets right

**It knows a chord from a word.** A line is only treated as chords when everything on it is a chord, so "Am I going home" stays a lyric even though it starts with `Am`.

**Chords stay in their columns.** `G` becoming `G#` is one character wider. capo takes that character back from the gap that follows, so the next chord still sits over the right word.

**It spells notes the way musicians do.** Moving to Bb gives you `Bb` and `Eb`, not `A#` and `D#`. Moving to A gives `F#m`, not `Gbm`.

**It reads both common formats.** Chords on their own line above the lyrics, and chords in brackets inside the lyrics:

```
[Am]Say it once and [C]say it slow
```

**It handles the awkward chords.** `F#m7`, `Bbmaj7`, `D/F#`, `Gsus4`, `C7b5`, `Edim`, bar lines and `x2` repeats.

## How the key is found

Every major key has six chords that belong to it. capo scores each of the twelve keys by how many of the song's chords belong, counts the first and last chord for more, and picks the best. If the song starts and ends on the relative minor, it calls it minor.

It is a guess, and a good one for ordinary songs. A song that changes key halfway will confuse it; use `capo up` with a number then.

## Run the tests

```bash
go test ./...
```

They cover chord parsing, spelling, column alignment, key detection, Nashville numbers, and a round trip: up five semitones and back down gives the exact sheet you started with.

## Why

Rewriting a chord chart by hand every time someone wants a different key is slow and easy to get wrong. It is a small problem with clear rules, which makes it a good one to solve properly.

## License

MIT
