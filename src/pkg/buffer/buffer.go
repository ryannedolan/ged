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
)

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
  Name string
  buffer *Buffer
  rows, cols int
  curi, curj int
  top, cur, mark *node
}

type Reader struct {
  head, tail *node
  pending bytes.Buffer
}

func EOL(c rune) bool {
  return c == '\n'
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

func (b *Buffer) Clear() {
  b.head, b.tail = nil, nil
}

func (b *Buffer) Window(name string, rows, cols int) *Window {
  return &Window{Name: name, buffer: b, top: b.head, cur: b.head, rows: rows, cols: cols}
}

func (w *Window) Write(p []byte) (int, error) {
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
    w.Insert(c)
  }
}

func (w *Window) NewReader() *Reader {
  first, last := w.buffer.head, w.buffer.tail
  if w.mark != nil {
    first, last = w.marked()
  }    
  return &Reader{head: first, tail: last}
}

func (r *Reader) Read(p []byte) (int, error) {
  if r.pending.Len() != 0 {
    return r.pending.Read(p)
  }
  if r.head == r.tail {
    return 0, io.EOF
  }
  for r.head != r.tail {
    r.pending.WriteRune(r.head.c)
    r.head = r.head.next
  }
  return r.pending.Read(p)
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
  b.Append('\n')
}

func (b *Buffer) NewLine() {
  b.Append('\n')
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

func (w *Window) InsertString(s string) {
  reader := strings.NewReader(s)
  for {
    c, _, err := reader.ReadRune()
    if err != nil {
      return
    }
    w.Insert(c)
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
  case '\r': w.Insert('\n')
  case 8, 0x7F: w.Backspace()
  default: return false
  }
  return true
}

func (w *Window) marked() (first, last *node) {
  if w.mark == nil {
    return
  }
  for pos := w.buffer.head; pos != nil; pos = pos.next {
    if pos == w.cur || pos == w.mark {
      if first == nil {
        first = pos
      } else if last == nil {
        last = pos
      } else {
        break
      }
    }
  }
  return
}

func (w *Window) forMarked(f func(p *node)) {
  first, last := w.marked()
  for pos := first; pos != last; pos = pos.next {
    f(pos)
  }
}

// Delete the rune at the cursor's position.
func (w *Window) Delete() {
  w.forMarked(func(p *node) {
    p.delete()
  })
  w.cur = w.cur.delete()
  w.mark = nil
}

func (w *Window) Render(ras *raster.Raster) {
  i := 0
  j := 0
  markDown := false
  for pos := w.top; pos != nil && i < w.rows; pos = pos.next {
    if pos == w.mark {
      markDown = !markDown
    }
    if pos == w.cur {
      w.curi, w.curj = i, j
      ras.Cursor(i, j)
      if w.mark != nil {
        markDown = !markDown
      }
    }
    if EOL(pos.c) {
      i++
      j = 0
    } else if pos.c == '\t' {
      j++
      for j % 8 != 0 {
        j++
      }
    } else if j < w.cols {
      style := raster.NORMAL
      if markDown {
        style = raster.HIGHLIGHT
      }
      ras.Put(i, j, pos.c, style)
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
func (p *node) seek(f func(c rune) bool) (pos *node, n int) {
  pos = p
  // move at least once
  if pos.next != nil {
    pos = pos.next
    n++
  }
  for pos.next != nil && !f(pos.prev.c) {
    pos = pos.next
    n++ 
  }
  return
}

// move pointer just past previous c
func (p *node) seekback(f func(c rune) bool) (pos *node, n int) {
  pos = p
  // move at least once
  if pos.prev != nil {
    pos = pos.prev
    n++
  }
  for pos.prev != nil && !f(pos.prev.c) {
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

func (p *node) delete() *node {
  if p.next != nil {
    p.next.prev = p.prev
  }
  if p.prev != nil {
    p.prev.next = p.next
  }
  p = p.next
  return p
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
  for w.cur.next != nil && !EOL(w.cur.c) {
    i++
    w.cur = w.cur.next
  }
  return
}

func (w *Window) Mark() {
  w.mark = w.cur
}

func (w *Window) ClearMark() {
  w.mark = nil
}

func (w *Window) MarkedText() string {
  var builder strings.Builder
  w.forMarked(func (p *node) {
    builder.WriteRune(p.c)
  })
  return builder.String()
}

func (w *Window) Yank() string {
  var builder strings.Builder
  w.forMarked(func (p *node) {
    builder.WriteRune(p.c)
    p.delete()
  })
  builder.WriteRune(w.cur.c)
  w.cur = w.cur.delete()
  return builder.String()
}

func (w *Window) Find(pat *regexp.Regexp) {
  //todo
}

func (w *Window) FindReverse(pat *regexp.Regexp) {
  //todo
}

func (w *Window) Plumb() {
}

