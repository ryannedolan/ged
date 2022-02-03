package rfs

import (
  "io/fs"
  "path/filepath"
  "time"
  "fmt"
  "golang.org/x/crypto/ssh"
)

type RFS struct {
  *ssh.Client
}

type RFile struct {
  RemotePath string
  fs *RFS
  pos int
}

type fileInfo struct {
  name string
  size int64
  mode fs.FileMode
  modTimeSeconds int64
  fs *RFS
}

func NewRFS(conn *ssh.Client) *RFS {
  return &RFS{conn}
}

func (fs *RFS) Open(remotePath string) (fs.File, error) {
  return &RFile{RemotePath: remotePath, fs: fs}, nil
}

func (f *RFile) Read(buf []byte) (int, error) {
  session, err := f.fs.NewSession()
  if err != nil {
    return 0, err
  }
  defer session.Close()
  r, err := session.StdoutPipe()
  if err != nil {
    return 0, err
  }
  err = session.Run(fmt.Sprintf("dd iflag=skip_bytes skip=%d count=1 if=%s", f.pos, f.RemotePath))
  if err != nil {
    return 0, err
  }
  i, err := r.Read(buf)
  f.pos += i
  return i, err
}

func (f RFile) Close() error {
  return nil
}

func (f *RFile) Stat() (fs.FileInfo, error) {
  var info fileInfo
  info.fs = f.fs
  info.name = filepath.Base(f.RemotePath)
  session, err := f.fs.NewSession()
  if err != nil {
    return info, err
  }
  defer session.Close()
  r, err := session.StdoutPipe()
  if err != nil {
    return info, err
  }
  err = session.Run(fmt.Sprintf("stat --format=\"%%s %%f %%Y\" %s", f.RemotePath))
  if err != nil {
    return info, err
  }
  _, err = fmt.Fscanf(r, "%d %x %d", &info.size, &info.mode, &info.modTimeSeconds)
  return info, err 
}

func (f fileInfo) Name() string {
  return f.name 
}

func (f fileInfo) Size() int64 {
  return f.size
}

func (f fileInfo) IsDir() bool {
  return f.mode & fs.ModeDir != 0
}

func (f fileInfo) Mode() fs.FileMode {
  return fs.FileMode(f.mode) 
}

func (f fileInfo) ModTime() time.Time {
  return time.Unix(f.modTimeSeconds, 0)
}

func (f fileInfo) Sys() interface{} {
  return f.fs
}
