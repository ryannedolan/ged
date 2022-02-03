// Stats arg[0] using stdin/out, which is expected to be a remote shell
package main

import (
  "../../src/pkg/rfs"
  "os"
  "io"
  "io/ioutil"
  "log"
  "fmt"
  "bufio"
  "golang.org/x/crypto/ssh"
)

func main() {
  name := "/dev/urandom"
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
  stat, _ := f.Stat()
  io.Copy(os.Stdout, bufio.NewReader(f))
  fmt.Println("name: ", stat.Name())
  fmt.Println("size: ", stat.Size())
  fmt.Println("time: ", stat.ModTime())
}
