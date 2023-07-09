package main

import (
	"errors"
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
)

func main() {
	switch os.Args[1] {
	case "run":
		run()
	case "child":
		child()
	default:
		panic("Bad Command.")
	}
}

func run() {
	fmt.Printf("Running: %v as %d \n", os.Args[2:], os.Getpid())
	cmd := exec.Command("/proc/self/exe", append([]string{"child"}, os.Args[2:]...)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags:   syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS,
		Unshareflags: syscall.CLONE_NEWNS, // to not share container's mountspace with the host
	}

	cmd.Run()
}

func child() {
	fmt.Printf("Running: %v as %d \n", os.Args[2:], os.Getpid())

	cg()

	syscall.Sethostname([]byte("container"))
	syscall.Chroot("./fs/") // root-jail
	syscall.Chdir("/")      //cd to root. Undefined working directory after chroot
	syscall.Mount("proc", "proc", "proc", 0, "")

	cmd := exec.Command(os.Args[2], os.Args[3:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.Run()

	syscall.Unmount("/proc", 0)
}

// Control Group
func cg() {
	cgdir := "/sys/fs/cgroup"
	pidir := filepath.Join(cgdir, "pids")

	err := os.Mkdir(filepath.Join(pidir, "acont"), 0755) //acontainer
	if err != nil && !errors.Is(err, fs.ErrExist) {
		panic(err)
	}
	// /sys/fs/cgroup/pids/...
	err = ioutil.WriteFile(filepath.Join(pidir, "acont/pids.max"), []byte("20"), 0700)
	if err != nil {
		panic(err)
	}

	// (Sends signal to processes in the cgroup) To remove the cgroup after container exits
	err = ioutil.WriteFile(filepath.Join(pidir, "acont/notify_on_release"), []byte("1"), 0700)
	if err != nil {
		panic(err)
	}

	// add process to cgroup as member
	err = ioutil.WriteFile(filepath.Join(pidir, "acont/cgroup.procs"), []byte(strconv.Itoa(os.Getpid())), 0700)
	if err != nil {
		panic(err)
	}
}
