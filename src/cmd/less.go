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
  "bufio"
  "golang.org/x/crypto/ssh"
  "golang.org/x/term"
)

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
  buf := buffer.New(f)
  ras := raster.New(25, 80)
  oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
  if err != nil {
    panic(err)
  }
  defer term.Restore(int(os.Stdin.Fd()), oldState)
  in := bufio.NewReader(os.Stdin)
  w := buf.Window(25, 80)
  for {
    w.Render(ras)
    io.Copy(os.Stdout, ras)
    rn, _, err := in.ReadRune()
    if err != nil {
      panic(err)
    }
    switch (rn) {
    case 'j': w.Down()
    case 'k': w.Up()
    case 'l': w.Right()
    case 'h': w.Left()
    case 'q': return
    }
  }
}
