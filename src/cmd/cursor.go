// Stats arg[0] using stdin/out, which is expected to be a remote shell
package main

import (
  "../../src/pkg/rfs"
  "../../src/pkg/raster"
  "../../src/pkg/buffer"
  "os"
  "io"
  "io/ioutil"
  "log"
  "fmt"
  "bufio"
  "regexp"
  "golang.org/x/crypto/ssh"
  "golang.org/x/term"
)

var leftBrackets = regexp.MustCompile("\\[|\\{|\\(")
var rightBrackets = regexp.MustCompile("\\]|\\}|\\)")

func main() {
  name := "/proc/cpuinfo"
  if len(os.Args) > 1 {
    name = os.Args[1]
  }
	key, err := ioutil.ReadFile("/home/ryanne/.ssh/test")
	if err != nil {
		log.Fatalf("unable to read private key: %v", err)
	}
	// Create the Signer for this private key.
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		log.Fatalf("unable to parse private key: %v", err)
	}
  config := ssh.ClientConfig{
    User: "ryanne",
		Auth: []ssh.AuthMethod{
			// Use the PublicKeys method for remote authentication.
			ssh.PublicKeys(signer),
		},
    HostKeyCallback: ssh.InsecureIgnoreHostKey(),
  }
  c, err := ssh.Dial("tcp", "localhost:22", &config)
  if err != nil {
    panic(err)
  }
  fs := rfs.NewRFS(c)
  f, _ := fs.Open(name)
  buf := buffer.FromFile(f, buffer.Config{TabWidth: 8})
  cols, rows, err := term.GetSize(int(os.Stdin.Fd()))
  if err != nil {
    panic(err)
  }
  ras := raster.New(rows, cols)
  oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
  if err != nil {
    panic(err)
  }
  defer term.Restore(int(os.Stdin.Fd()), oldState)
  in := bufio.NewReader(os.Stdin)
  w := buf.Window(25, 80)
  mode := 'x'
  clip := ""
  for {
    w.Render(ras)
    io.Copy(os.Stdout, ras)
    rn, _, err := in.ReadRune()
    if err != nil {
      panic(err)
    }
    if rn == '\033' {
      mode = 'x'
      w.ClearMark()
    } else if mode == 'i' {
      w.Insert(rn)
    } else if mode == 'o' {
      w.Overwrite(rn)
    } else {
      switch (rn) {
      case 'i': mode = 'i'
      case 'o': mode = 'o'
      case 'v': mode = 'v'; w.Mark()
      case 'h': w.Left()
      case 'j': w.Down()
      case 'k': w.Up()
      case 'l': w.Right()
      case 'x','d': w.Delete()
      case '0': w.Home()
      case '$': w.End()
      case 'A': w.End(); mode = 'i'
      case '[': w.FindReverse(leftBrackets)
      case ']': w.Find(rightBrackets)
      case 13:  w.Plumb()
      case 'y': clip = w.Yank()
      case 'p': w.InsertString(clip)
      case 'q': return
      case '\033': return
      default: fmt.Print("\a")
      }
    }
  }
}
