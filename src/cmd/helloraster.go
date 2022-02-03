package main

import (
  "../../src/pkg/raster"
  "os"
  "io"
  "fmt"
  "time"
  "math/rand"
)

func main() {
  ras := raster.New(25, 80)
  start := time.Now()
  for i := 0; i < 10000; i++ {
    ras.Clear()
    for j := 0; j < 1000; j++ {
      ras.PutString(rand.Intn(24), rand.Intn(79), "hello world!")
    }
    ras.PutString(0, 0, fmt.Sprintf("Frame Rate: [%d]", int(float64(i) / float64(time.Since(start).Seconds()))))
    io.Copy(os.Stdout, ras)
  }
}
