package k8s

import (
	"bytes"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/kubeedge/kubeedge/tests/e2e/utils"
)

func CallSysCommand(command string) (string, error) {
	cmd := exec.Command("/bin/sh", "-c", command)
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		err = fmt.Errorf("%s: %s", err, stderr.String())
		return "", err
	}
	return out.String(), nil
}

func FetchIPFromDigOutput(outStr, domain string) string {
	// delete space and tab
	outStr = strings.Replace(outStr, " ", "", -1)
	outStr = strings.Replace(outStr, "\t", "", -1)

	flagStr := "INA"
	index := strings.Index(outStr, flagStr)
	if index == -1 {
		return ""
	}
	index2 := strings.Index(outStr[index:], "\n")
	ip := outStr[index+len(flagStr) : index+index2]
	return ip
}

func FetchHostnameAndStatusCodeFromOutput(outStr string) (hostname string, statusCode int) {
	parts := strings.Split(outStr, "\n")
	if len(parts) != 2 {
		return
	}
	hostname = parts[0]
	statusCode, err := strconv.Atoi(parts[1])
	if err != nil {
		utils.Fatalf("FetchHostnameAndStatusCodeFromOutput strconv failed")
		return
	}
	return
}

func FetchTCPReplyFromOutput(outStr string) string {
	return strings.TrimSpace(outStr)
}
