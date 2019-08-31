package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	stdioFdCount = 3
	myname       = "./mypipe"
	pMessage     = "msg-from-parent"
	cMessage     = "msg-from-child"
)

func Log(format string, args ...interface{}) {
	f, err := os.OpenFile(fmt.Sprintf("%s-%s.log", myname, time.Now().Format("2006-01-02-15")), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	f.WriteString(fmt.Sprintf(time.Now().Format("2006-01-02 15:04:05 ")+format+"\n", args...))
}

func NewSockPair(name string) (parent *os.File, child *os.File, err error) {
	fds, err := syscall.Socketpair(syscall.AF_LOCAL, syscall.SOCK_STREAM|syscall.SOCK_CLOEXEC, 0)
	if err != nil {
		return nil, nil, err
	}
	return os.NewFile(uintptr(fds[1]), name+"-p"), os.NewFile(uintptr(fds[0]), name+"-c"), nil
}

func RunParent(name string) error {
	pipep, pipec, err := NewSockPair(name)
	if err != nil {
		Log("new pipe %s failed %v", name, err)
		return err
	}
	defer pipep.Close()
	cmd := exec.Command(myname, fmt.Sprintf("child-%s", name[6:]))
	cmd.ExtraFiles = append(cmd.ExtraFiles, pipec)
	cmd.Env = append(cmd.Env, fmt.Sprintf("_LIBCONTAINER_INITPIPE=%d", stdioFdCount+len(cmd.ExtraFiles)-1))
	if err := cmd.Start(); err != nil {
		Log("cmd start %s failed %v", name, err)
		return err
	}
	pipec.Close()

	data := make([]byte, 4096)
	if _, err := pipep.Write([]byte(pMessage)); err != nil {
		Log("%s write pipe failed %v", name, err)
	}
	if _, err := pipep.Read(data); err != nil {
		Log("%s read pipe failed %v", name, err)
	}
	if err := syscall.Shutdown(int(pipep.Fd()), syscall.SHUT_WR); err != nil {
		Log("shutdown pipe %s failed %v", name, err)
	}
	cmd.Wait()
	return nil
}

func RunChild(name string) error {
	envInitPipe := os.Getenv("_LIBCONTAINER_INITPIPE")
	pipefd, err := strconv.Atoi(envInitPipe)
	if err != nil {
		return fmt.Errorf("unable to convert _LIBCONTAINER_INITPIPE=%s to int: %s", envInitPipe, err)
	}
	pipe := os.NewFile(uintptr(pipefd), "pipe")
	defer pipe.Close()
	data := make([]byte, 4096)
	if _, err = pipe.Read(data); err != nil {
		Log("%s child read pipe failed", name, err)
	}
	if _, err = pipe.Write([]byte(cMessage)); err != nil {
		Log("%s child write pipe failed %v", name, err)
	}
	return nil
}

func loop(n int) {
	for i := 0; i < n; i++ {
		name := fmt.Sprintf("parent-%d-%d", n, i)
		fmt.Printf("loop %s beg ---------------------------\n", name)
		cmd := exec.Command(myname, name)
		cmd.Start()
		cmd.Wait()
		fmt.Printf("loop %s end ---------------------------\n", name)
	}

	return
}

func main() {
	args := len(os.Args)
	if args != 2 {
		loop(1)
		return
	}

	subcmd := os.Args[1]
	Log("%s started ...", subcmd)
	defer Log("%s done", subcmd)
	if strings.Contains(subcmd, "parent") {
		err := RunParent(subcmd)
		if err != nil {
			Log("parent error %v", err)
		}
	}
	if strings.Contains(subcmd, "child") {
		err := RunChild(subcmd)
		if err != nil {
			Log("child error %v", err)
		}
	}
}
