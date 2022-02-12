// A list of lines that can be edited and rendered.
package buffer

import (
  "io"
  "io/fs"
//  "bufio"
  "strings"
  "../raster"
  "regexp"
  "bytes"
//  "fmt"
  "os/exec"
)

const EOL = '\n'

type node struct {
  c rune
  prev, next *node
}

type Config struct {
  TabWidth int
}

type Buffer struct {
  head, tail *node
  Config Config
  pending bytes.Buffer
}

type Window struct {
  buffer *Buffer
  rows, cols int
  curi, curj int
  top, cur, mark *node
}

func FromFile(f fs.File, config Config) *Buffer {
  b := &Buffer{Config: config}
  io.Copy(b, f)
  return b
}

func (b *Buffer) Read(p []byte) (int, error) {
  if b.pending.Len() != 0 {
    return b.pending.Read(p)
  }
  for b.head != nil {
    b.pending.WriteRune(b.head.c)
    b.head = b.head.next
  }
  return b.pending.Read(p)
}

func (b *Buffer) Write(p []byte) (int, error) {
  reader := bytes.NewBuffer(p)
  count := 0
  for {
    c, n, err := reader.ReadRune()
    count += n 
    if n == 0 {
      return count, nil
    } else if err != nil {
      return count, err
    }
    b.Append(c)
  }
}

func (b *Buffer) Window(rows, cols int) *Window {
  return &Window{buffer: b, top: b.head, cur: b.head, rows: rows, cols: cols}
}

func (b *Buffer) Append(c rune) {
  p := &node{c: c, prev: b.tail}
  if b.tail != nil {
    b.tail.next = p
  }
  if b.head == nil {
    b.head = p
  }
  b.tail = p
}

func (b *Buffer) AppendString(s string) {
  for _, c := range s {
    b.Append(c)
  }
}

func (b *Buffer) AppendLine(s string) {
  b.AppendString(s)
  b.Append(EOL)
}

func (b *Buffer) NewLine() {
  b.Append(EOL)
}

func (b *Buffer) WriteString(s string) (int, error) {
  for _, c := range s {
    b.Append(c)
  }
  return len(s), nil
}

func (b Buffer) String() string {
  var builder strings.Builder
  for pos := b.head; pos != nil; pos = pos.next {
    builder.WriteRune(pos.c)
  }
  return b.String()
}

// Insert a rune at the cursor's position.
func (w *Window) Insert(c rune) {
  if w.handleKeys(c) {
    return
  }
  p := &node{c: c}
  if w.cur == nil {
    w.cur = w.buffer.tail
  }
  if w.cur == w.buffer.tail {
    w.buffer.Append(c)
  } else if w.cur == w.buffer.head {
    //w.buffer.Prepend(c)
  } else {
    p.next = w.cur
    p.prev = w.cur.prev
    if w.cur.prev != nil {
      w.cur.prev.next = p
    }
    w.cur.prev = p
  }
}

func (w *Window) Backspace() {
  if w.cur.prev != nil {
    w.cur.prev.next = w.cur.next
  }
  if w.cur.next != nil {
    w.cur.next.prev = w.cur.prev
  }
  w.cur = w.cur.prev
}

func (w *Window) Overwrite(c rune) {
  if w.handleKeys(c) {
    return
  }
}

func (w *Window) handleKeys(c rune) bool {
  switch (c) {
  case '\r': w.Insert(EOL)
  case 8, 0x7F: w.Backspace()
  default: return false
  }
  return true
} 

// Delete the rune at the cursor's position.
func (w *Window) Delete() {
  if w.cur.next != nil {
    w.cur.next.prev = w.cur.prev
  }
  if w.cur.prev != nil {
    w.cur.prev.next = w.cur.next
  }
  w.cur = w.cur.next
}

func (w *Window) Render(ras *raster.Raster) {
  i := 0
  j := 0
  for pos := w.top; pos != nil && i < w.rows; pos = pos.next {
    if pos == w.cur {
      w.curi, w.curj = i, j
      ras.Cursor(i, j)
    }
    if pos.c == EOL {
      i++
      j = 0
    } else if pos.c == '\t' {
      j++
      for j % 8 != 0 {
        j++
      }
    } else if j < w.cols {
      ras.Put(i, j, pos.c, raster.NORMAL)
      j++
    }
  }
}

func (w *Window) Right() {
  if w.cur != nil && w.cur.next != nil {
    w.cur = w.cur.next 
  }
}

func (w *Window) Left() {
  if w.cur != nil && w.cur.prev != nil {
    w.cur = w.cur.prev
  }
}

func (w *Window) Up() {
  col := w.Home()
  w.Left()
  w.Home()
  for i := 0; i < col; i++ {
    w.Right()
  }
  if w.curi == 0 {
    w.ScrollUp()
  }
}

func (w *Window) Down() {
  col := w.Home()
  w.End()
  w.Right()
  for i := 0; i < col; i++ {
    w.Right()
  }
  if w.curi == w.rows - 1 {
    w.ScrollDown()
  }
}

// move pointer just past next c
func (p *node) seek(c rune) (pos *node, n int) {
  pos = p
  // move at least once
  if pos.next != nil {
    pos = pos.next
    n++
  }
  for pos.next != nil && pos.prev.c != c {
    pos = pos.next
    n++ 
  }
  return
}

// move pointer just past previous c
func (p *node) seekback(c rune) (pos *node, n int) {
  pos = p
  // move at least once
  if pos.prev != nil {
    pos = pos.prev
    n++
  }
  for pos.prev != nil && pos.prev.c != c {
    pos = pos.prev
    n++
  }
  return
}

func (p *node) skip(n int) (pos *node) {
  pos = p
  for i := 0; i < n && pos.next != nil; i++ {
    pos = pos.next
  }
  return
}

func (p *node) skipback(n int) (pos *node) {
  pos = p 
  for i := 0; i < n && pos.prev != nil; i++ {
    pos = pos.prev
  }
  return
}

func (w *Window) ScrollDown() {
  w.top, _ = w.top.seek(EOL)
}

func (w *Window) ScrollUp() {
  w.top, _ = w.top.seekback(EOL) 
}

func (w *Window) PageDown() {
  for i := 0; i < w.rows; i++ {
    w.ScrollDown()
  }
}

func (w *Window) PageUp() {
  for i := 0; i < w.rows; i++ {
    w.ScrollUp()
  }
}

func (w *Window) Home() (n int) {
  w.cur, n = w.cur.seekback(EOL) 
  return
}

func (w *Window) End() (n int) {
  i := 0
  for w.cur.next != nil && w.cur.c != EOL {
    i++
    w.cur = w.cur.next
  }
  return
}

func (w *Window) Mark() {
  w.mark = w.cur
}

func (w *Window) MarkedText() string {
  return "todo"
}

func (w *Window) Find(pat *regexp.Regexp) {
  //todo
}

func (w *Window) FindReverse(pat *regexp.Regexp) {
  //todo
}

func (w *Window) Run(in *Buffer) *Buffer {
  out := &Buffer{Config: in.Config}
  cmd := exec.Command("/bin/sh", "-c", w.buffer.String())
  if err := cmd.Run(); err != nil {
    //fmt.Fprintf(out, "ERROR: %e", err)
  }
  return out
}
