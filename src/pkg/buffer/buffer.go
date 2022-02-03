// A list of lines that can be edited and rendered.
package buffer

import (
  "io/fs"
  "bufio"
  "strings"
  "../raster"
)

type Line struct {
  chars []rune
  prev *Line
  next *Line
}

type Buffer struct {
  head, tail *Line
}

type Window struct {
  top *Line
  rows, cols int
  offset int
  curi, curj int
}

func New(f fs.File) *Buffer {
  scanner := bufio.NewScanner(f)
  b := &Buffer{}
  for scanner.Scan() {
    b.Append(scanner.Text())
  }
  return b
}

func (b *Buffer) Window(rows, cols int) *Window {
  return &Window{top: b.head, rows: rows, cols: cols}
}

func (b *Buffer) Append(s string) {
  s = strings.ReplaceAll(s, "\t", "  ")
  line := &Line{chars: []rune(s), prev: b.tail}
  if b.tail != nil {
    b.tail.next = line
  }
  b.tail = line
  if b.head == nil {
    b.head = line
  }
}

func (l Line) String() string {
  return strings.TrimRight(string(l.chars), " ")
}

func (b Buffer) String() string {
  var builder strings.Builder
  for pos := b.head; pos != nil; pos = pos.next {
    builder.WriteString(pos.String())
  }
  return b.String()
}

// Insert a rune at the cursor's position.
func (w *Window) Insert(c rune) {
  line := w.Line()
  if line == nil {
    return
  }
  j := w.curj
  for len(line.chars) < j {
    line.chars = append(line.chars, ' ')
  }
  if j < len(line.chars) {
    prefix := append(line.chars[:j], c)
    suffix := line.chars[j:]
    line.chars = append(prefix, suffix...)
  } else {
    line.chars = append(line.chars, c)
  }
  w.CursorRight()
}

func (w *Window) Overwrite(c rune) {
  line := w.Line()
  if line == nil {
    return
  }
  j := w.curj
  for len(line.chars) < j {
    line.chars = append(line.chars, ' ')
  }
  if j < len(line.chars) {
    line.chars[w.curj] = c
  } else {
    line.chars = append(line.chars, c)
  }
  w.CursorRight()
}


// Delete the rune at the cursor's position.
func (w *Window) Delete() {
  line := w.Line()
  if line == nil {
    return
  }
  j := w.curj
  linelen := len(line.chars)
  if j < linelen {
    line.chars = append(line.chars[:j], line.chars[j + 1:] ...)
  } else if linelen > 0 {
    line.chars = line.chars[:linelen -1]
  }
}

func (w *Window) Line() (line *Line) {
  line = w.top
  for i := 0; i < w.curi; i++ {
    if line != nil && line.next != nil {
      line = line.next
    }  
  } 
  return
}

func (w *Window) DownN(n int) {
  for i := 0; i < n; i++ {
    w.Down()
  }
}

func (w *Window) Down() {
  if w.top.next != nil {
    w.top = w.top.next
  }
}

func (w *Window) UpN(n int) {
  for i :=0; i < n; i++ {
    w.Up()
  }
}

func (w *Window) Up() {
  if w.top.prev != nil {
    w.top = w.top.prev
  }
}

func (w *Window) Right() {
  w.offset++
}

func (w *Window) Left() {
  if w.offset > 0 {
    w.offset--
  }
}

func (w *Window) Render(ras *raster.Raster) {
  ras.Cursor(w.curi, w.curj)
  i := 0
  for pos := w.top; pos != nil && i < w.rows; pos = pos.next {
    ras.ClearLine(i)
    if w.offset < len(pos.chars) {
      ras.PutString(i, 0, w.offset, string(pos.chars))
    }
    i++
  }
  for i < w.rows {
    ras.ClearLine(i)
    i++
  }
}

func (w *Window) CursorRight() {
  if w.curj >= w.cols - 1 {
    w.curj = w.cols - 1
    w.Right()
  } else {
    w.curj++
  }
}

func (w *Window) CursorLeft() {
  if w.curj <= 0 {
    w.curj = 0
    w.Left()
  } else {
    w.curj--
  }
}

func (w *Window) CursorUp() {
  if w.curi <= 0 {
    w.curi = 0
    w.Up()
  } else {
    w.curi--
  }
}

func (w *Window) CursorDown() {
  if w.curi >= w.rows - 1 {
    w.curi = w.rows - 1
    w.Down()
  } else {
    w.curi++
  }
}

func (w *Window) CursorHome() {
  w.curj = 0
  w.offset = 0
}

func (w *Window) CursorEnd() {
  w.curj = len(string(w.Line().chars))
  if w.curj > w.cols {
    w.offset += w.curj - w.cols
    w.curj = w.cols - 1
  }
}
