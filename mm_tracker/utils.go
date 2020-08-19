package main

import (
	"io/ioutil"
	"log"
	"os/exec"
	"strings"
)

// Takes command and argument as slice and returns stdout and stderr
func execUserCmd(userCmdArgs []string) (string, string, error) {

	//log.Println("Executing:", strings.Join(userCmdArgs, " "))

	cmd := exec.Command(userCmdArgs[0])
	cmd.Args = userCmdArgs

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", "", err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", "", err
	}
	cmd.Start()
	output, _ := ioutil.ReadAll(stdout)
	errout, _ := ioutil.ReadAll(stderr)
	cmd.Wait()

	return string(output), string(errout), nil
}

func execShellCmdInternal(cmd string, print bool) string {
	cmdArgs := strings.Split(cmd, " ")
	stdout, errout, _ := execUserCmd(cmdArgs)
	if stdout != "" && print {
		log.Println(stdout)
	}
	if errout != "" {
		log.Println("error is:", errout)
	}
	return stdout
}

func execShellCmd(cmd string) {
	execShellCmdInternal(cmd, true)
}

func execShellCmdOutput(cmd string) string {
	return execShellCmdInternal(cmd, false)
}
