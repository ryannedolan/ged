package raster

import (
  "bytes"
  "fmt"
  "strings"
)

// Keeps track of what has been rendered already, s.t. writes to the screen are minimized.
type Raster struct {
  rows, cols int
  chars [][]rune
  dirty []bool
  pending bytes.Buffer
  curi, curj int
}

func New(rows, cols int) *Raster {
  chars := make([][]rune, rows)
  for i := 0; i < rows; i++ {
    chars[i] = make([]rune, cols)
    for j := 0; j < cols; j++ {
      chars[i][j] = ' '
    }
  }
  return &Raster{rows, cols, chars, make([]bool, rows), bytes.Buffer{}, 0, 0}
}

func (r *Raster) Cursor(i, j int) {
  if r.curi != i || r.curj != j {
    r.dirty[i] = true
  }
  r.curi = i
  r.curj = j 
}

func (r *Raster) Put(i, j int, c rune) {
  r.dirty[i] = true
  r.chars[i][j] = c
}

// Writes the string at the given location. The string is not wrapped.
func (r *Raster) PutString(i, j, offset int, s string) {
  if i >= len(r.chars) {
    return
  }
  if len(s) == 0 {
    return
  }
  r.dirty[i] = true
  reader := strings.NewReader(s)
  for j < r.cols {
    c, sz, err := reader.ReadRune()
    if sz == 0 {
      return
    }
    if c == '\t' {
      c = ' '
    }
    if offset > 0 {
      // read runes but don't write them until we hit the offset
      offset -= 1
    } else { 
      r.chars[i][j] = c 
      j++
    }
    if err != nil {
      return
    }
  }
}

// Writes the string at the given location. The string is repeated.
func (r *Raster) PutLine(i int, s string) {
  // TODO this is terrible
  for len(s) < len(r.chars[i]) {
    s = s + s
  }
  r.PutString(i, 0, 0, s)
}

func (r *Raster) ClearLineWith(i int, b rune) {
  r.dirty[i] = true
  for j := 0; j < r.cols; j++ {
    r.chars[i][j] = b
  }
}

func (r *Raster) ClearLine(i int) {
  r.ClearLineWith(i, ' ')
}

func (r *Raster) ClearWith(b rune) {
  for i := 0; i < r.rows; i++ {
    r.dirty[i] = true
    for j := 0; j < r.cols; j++ {
      r.chars[i][j] = b
    }
  }
}

func (r *Raster) Clear() {
  r.ClearWith(' ')
}

// Reads any dirty lines. Callers should tail this stream to get continuous updates.
func (r *Raster) Read(buf []byte) (int, error) {
  // flush any pending reads, in case buf is too small
  if r.pending.Len() != 0 {
    return r.pending.Read(buf)
  }

  for i := 0; i < len(r.chars); i++ {
    if r.dirty[i] {
      r.dirty[i] = false
      line := string(r.chars[i])
      if r.curi == i {
        // draw line with cursor
        prefix := ""
        if r.curj > 0 && r.curj < len(line) {
          prefix = line[:r.curj]
        }
        suffix := " "
        if r.curj + 1 < len(line) {
          suffix = line[r.curj + 1:]
        }
        c := line[r.curj]
        fmt.Fprintf(&r.pending, "\033[4m\033[%d;1H%s\033[7m%c\033[0m\033[4m%s\033[0m", i + 1, prefix, c, suffix)
      } else {
        fmt.Fprintf(&r.pending, "\033[0m\033[%d;1H%s", i + 1, string(r.chars[i]))
      }
    }
  }
  return r.pending.Read(buf)
}
