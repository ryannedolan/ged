package main

import (
  "../../src/pkg/rfs"
  "../../src/pkg/rexec"
  "../../src/pkg/raster"
  "../../src/pkg/buffer"
  "os"
  "io"
  "io/ioutil"
  "log"
  "fmt"
  "bufio"
  "bytes"
//  "regexp"
  "golang.org/x/crypto/ssh"
  "golang.org/x/term"
)

var oldState *term.State
var client *ssh.Client

func readLine(prompt string) string {
  term.Restore(int(os.Stdin.Fd()), oldState)
  _, rows, _ := term.GetSize(int(os.Stdin.Fd()))
  fmt.Printf("\033[%d;1H\033[K\033[31m%s", rows + 1, prompt)
  line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
  oldState, _ = term.MakeRaw(int(os.Stdin.Fd()))
  return line
}

func remoteShell(w *buffer.Window, line string) (buf *bytes.Buffer) {
  buf = new(bytes.Buffer)
  ros := rexec.NewROS(client)   
  cmd, err := ros.Command(line)
  if err != nil {
    showError(err)
    return
  }
  inf, _ := cmd.StdinPipe()
  outf, _ := cmd.StdoutPipe()
  errf, _ := cmd.StderrPipe()
  if err := cmd.Start(); err != nil {
    showError(err)
    return
  }
  io.Copy(inf, w.NewReader())
  inf.Close()
  errs, _ := ioutil.ReadAll(errf)
  if len(errs) > 0 {
    showError(fmt.Errorf("%s", errs))
    return
  }
  io.Copy(buf, outf)
  if err := cmd.Wait(); err != nil {
    showError(err)
    return
  }
  return
}

func showError(err error) {
  readLine(fmt.Sprintf("%s", err))
}

func showMsg(s string) {
  readLine(s)
}

func show(s string) {
  // TODO spawn `more` pager
  showMsg(s)
}

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
  client = c
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
  oldState, err = term.MakeRaw(int(os.Stdin.Fd()))
  if err != nil {
    panic(err)
  }
  defer term.Restore(int(os.Stdin.Fd()), oldState)
  in := bufio.NewReader(os.Stdin)
  w := buf.Window(name, 25, 80)
  mode := 'x'
  clip := ""
  for {
    w.Render(ras)
    io.Copy(os.Stdout, ras)
    rn, _, err := in.ReadRune()
    if err != nil {
      panic(err)
    }
    if rn == ':' {
      show(remoteShell(w, readLine(":")).String())
    } else if rn == '>' {
      w.InsertString(remoteShell(w, readLine(">")).String())
    } else if rn == '\033' {
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
