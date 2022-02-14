package rexec

import (
  "golang.org/x/crypto/ssh"
)

type ROS struct {
  *ssh.Client
}

type RCmd struct {
  cmd string 
  os *ROS
  *ssh.Session
}

func NewROS(conn *ssh.Client) *ROS {
  return &ROS{conn}
}

func (os *ROS) Command(cmd string) (*RCmd, error) {
  sess, err := os.NewSession()
  return &RCmd{cmd, os, sess}, err
}

func (r *RCmd) Run() error {
  return r.Session.Run(r.cmd)
}

func (r *RCmd) Start() error {
  return r.Session.Start(r.cmd)
}

