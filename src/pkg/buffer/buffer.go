// A list of lines that can be edited and rendered.
package buffer

import (
  "io"
  "io/fs"
  "bufio"
  "strings"
  "../raster"
  "unicode"
  "regexp"
  "bytes"
  "fmt"
  "os/exec"
)

type Node struct {
  c rune
  next *Node
}

type Line struct {
  head *Node
  tail *Node
  prev *Line
  next *Line
  size int
}

type Config struct {
  TabWidth int
}

type Buffer struct {
  head, tail *Line
  Config Config
}

type Window struct {
  buffer *Buffer
  top *Line
  rows, cols int
  offset int
  curi, curj int
  mark *Node
}

type BufferReader struct {
  buffer *Buffer
  pos *Line
  pending bytes.Buffer
}

func FromFile(f fs.File, config Config) *Buffer {
  scanner := bufio.NewScanner(f)
  b := &Buffer{Config: config}
  for scanner.Scan() {
    b.load(scanner.Text())
  }
  return b
}

func (b *Buffer) Window(rows, cols int) *Window {
  return &Window{buffer: b, top: b.head, rows: rows, cols: cols}
}

func (b *Buffer) load(s string) {
  line := toLine(s, b.Config.TabWidth)
  line.prev = b.tail
  if b.tail != nil {
    b.tail.next = line
  }
  b.tail = line
  if b.head == nil {
    b.head = line
  }
}

func toLine(s string, tabwidth int) (*Line) {
  r := strings.NewReader(s)
  line := &Line{}
  for {
    c, _, err := r.ReadRune()
    if err != nil {
      break
    }
    if c == '\t' {
      line.Append(' ')
      for line.size % tabwidth != 0 {
        line.Append(' ')
      }
    } else {
      line.Append(c)
    }
  }
  return line 
}

func (line *Line) Append(c rune) {
  if c == '\t' {
    panic("tried to append a tab")
  }
  if !unicode.IsGraphic(c) {
    c = ' '
  }
  node := &Node{c: c}
  if line.head == nil {
    line.head = node
  }
  if line.tail != nil {
    line.tail.next = node
  }
  line.tail = node
  line.size++
}

func (l *Line) String() string {
  var builder strings.Builder
  for pos := l.head; pos != nil; pos = pos.next {
    builder.WriteRune(pos.c)
  }
  return builder.String()
}

func (l *Line) Len() (n int) {
  for pos := l.head; pos != nil; pos = pos.next {
    n++
  }
  return
}

func (b Buffer) String() string {
  var builder strings.Builder
  for pos := b.head; pos != nil; pos = pos.next {
    builder.WriteString(pos.String())
  }
  return b.String()
}

func (b *Buffer) NewReader() *BufferReader {
  return &BufferReader{buffer: b, pos: b.head}
}

func (r *BufferReader) Read(p []byte) (int, error) {
  if r.pending.Len() == 0 {
    if r.pos == nil {
      return 0, io.EOF
    }
    fmt.Println(&r.pending, r.pos.String())
    r.pos = r.pos.next
  }
  return r.pending.Read(p)
}

func (b *Buffer) Write(p []byte) (int, error) {
  b.load(string(p))
  return len(p), nil
}

func (w *Window) handleNonprintable(c rune) bool {
  if c == 8 || c == 0x7F {
    w.Backspace()
    return true
  }
  if c == '\t' {
    w.Insert(' ')
    for (w.curj + w.offset) % w.buffer.Config.TabWidth != 0 {
      w.Insert(' ')
    }
    return true
  }
  return !unicode.IsGraphic(c)
}

// Insert a rune at the cursor's position.
func (w *Window) Insert(c rune) {
  if w.handleNonprintable(c) {
    return
  }
  line := w.Line()
  if line == nil {
    return
  }
  prev := line.head
  pos := line.head
  for j := 0; j < w.curj + w.offset; j++ {
    if pos == nil {
      line.Append(' ')
      pos = line.tail
    }
    prev = pos
    pos = pos.next
  }
  if prev == nil { 
    line.Append(c)
  } else {
    prev.next = &Node{c, pos}
  }
  w.CursorRight()
}

func (w *Window) Backspace() {
  w.CursorLeft()
  w.Delete()
}

func (w *Window) Overwrite(c rune) {
  if w.handleNonprintable(c) {
    return
  }
  line := w.Line()
  if line == nil {
    return
  }
  pos := line.head
  for j := 0; j < w.curj + w.offset; j++ {
    if pos == nil {
      line.Append(' ')
      pos = line.tail
    } else {
      pos = pos.next
    }
  }
  pos.c = c
  w.CursorRight()
}

// Delete the rune at the cursor's position.
func (w *Window) Delete() {
  line := w.Line()
  if line == nil {
    return
  }
  prev := line.head
  pos := line.head
  for j := 0; j < w.curj + w.offset; j++ {
    prev = pos
    if pos == nil {
      line.Append(' ')
      pos = line.tail
    } else {
      pos = pos.next
    }
  }
  if pos != nil {
    prev.next = pos.next
  } else {
    prev.next = nil
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
  i := 0
  for pos := w.top; pos != nil && i < w.rows; pos = pos.next {
    ras.ClearLine(i)
    pos2 := pos.head
    // skip until offset
    for j := 0; j < w.offset && pos2 != nil; j++ {
      pos2 = pos2.next
    }
    for j := 0; j < w.cols && pos2 != nil; j++ {
      if i == w.curi && j + w.offset == w.curj {
        ras.Put(i, j, pos2.c, 2)
      } else if i == w.curi {
        ras.Put(i, j, pos2.c, 1)
      } else {
        ras.Put(i, j, pos2.c, 0)
      }
      pos2 = pos2.next
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
  w.curj = w.Line().Len()
  for w.curj > w.cols {
    w.curj--
    w.offset++
  }
}

func (w *Window) Mark() {
}

func (w *Window) MarkedText() string {
  return "todo"
}

func (w *Window) Find(pat *regexp.Regexp) {
  oldi := w.curi
  for pos := w.Line(); pos != nil; pos = pos.next {
    if pat.MatchString(pos.String()) {
      return
    }
    w.CursorDown()
  }
  w.curi = oldi
}

func (w *Window) FindReverse(pat *regexp.Regexp) {
  oldi := w.curi
  for pos := w.Line(); pos != nil; pos = pos.prev {
    if pat.MatchString(pos.String()) {
      return
    }
    w.CursorUp()
  }
  w.curi = oldi
}

func (w *Window) Run(in *Buffer) *Buffer {
  out := &Buffer{Config: in.Config}
  cmd := exec.Command("/bin/sh", "-c", w.buffer.String())
  cmd.Stdin = in.NewReader()
  cmd.Stdout = out
  cmd.Stderr = w.buffer
  if err := cmd.Run(); err != nil {
    fmt.Fprintf(out, "ERROR: %e", err)
  }
  return out
}
