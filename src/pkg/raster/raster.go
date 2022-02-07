package raster

import (
  "bytes"
  "fmt"
  "strings"
)

type Style int

type char struct {
  c rune 
  style Style
}

// Keeps track of what has been rendered already, s.t. writes to the screen are minimized.
type Raster struct {
  rows, cols int
  chars [][]char
  dirty []bool
  pending bytes.Buffer
}

func New(rows, cols int) *Raster {
  chars := make([][]char, rows)
  for i := 0; i < rows; i++ {
    chars[i] = make([]char, cols)
    for j := 0; j < cols; j++ {
      chars[i][j] = char{}
    }
  }
  return &Raster{rows, cols, chars, make([]bool, rows), bytes.Buffer{}}
}

func (r *Raster) Put(i, j int, c rune, style Style) {
  r.dirty[i] = true
  r.chars[i][j] = char{c, style}
}

// Writes the string at the given location. The string is not wrapped.
func (r *Raster) PutString(i, j, offset int, s string, style Style) {
  if i >= r.rows {
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
      r.chars[i][j] = char{c, style} 
      j++
    }
    if err != nil {
      return
    }
  }
}

func (r *Raster) ClearLineWith(i int, b rune) {
  r.dirty[i] = true
  for j := 0; j < r.cols; j++ {
    r.chars[i][j] = char{b, 0}
  }
}

func (r *Raster) ClearLine(i int) {
  r.ClearLineWith(i, ' ')
}

func (r *Raster) ClearWith(b rune) {
  for i := 0; i < r.rows; i++ {
    r.dirty[i] = true
    for j := 0; j < r.cols; j++ {
      r.chars[i][j] = char{b, 0}
    }
  }
}

func (r *Raster) Clear() {
  r.ClearWith(0)
}

// Reads any dirty lines. Callers should tail this stream to get continuous updates.
func (r *Raster) Read(buf []byte) (int, error) {
  // flush any pending reads, in case buf is too small
  if r.pending.Len() != 0 {
    return r.pending.Read(buf)
  }

  styles := []string{"\033[0m", "\033[4m", "\033[7m"}
  prevStyle := Style(-1)
  for i := 0; i < r.rows; i++ {
    if r.dirty[i] {
        fmt.Fprintf(&r.pending, "\033[%d;1H", i + 1)
        line := r.chars[i]
        for _, c := range line {
          if c.style != prevStyle {
            r.pending.WriteString(styles[c.style])
            prevStyle = c.style
          }
          r.pending.WriteRune(c.c)
        }
      r.dirty[i] = false
    }
  }
  return r.pending.Read(buf)
}
