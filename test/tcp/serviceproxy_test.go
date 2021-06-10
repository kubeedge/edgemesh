package functions

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
)

func getProtocol(svcPorts string, port int) (string, string) {
	var protoName string
	sub := strings.Split(svcPorts, "|")
	n := len(sub)
	if n < 2 {
		return "", ""
	}
	svcName := sub[n-1]

	pstr := strconv.Itoa(port)
	if pstr == "" {
		return "", ""
	}
	for _, s := range sub {
		if strings.Contains(s, pstr) {
			protoName = strings.Split(s, ",")[0]
			break
		}
	}
	return protoName, svcName
}

func TestGetProtocol(t *testing.T) {
	svcPorts := "mqtt,1883|svc-mosquitto"
	port := 1883

	protoName, svcName := getProtocol(svcPorts, port)
	fmt.Printf("protoName:%s, svcName:%s\n", protoName, svcName)
}

func TestGetEpProtocol(t *testing.T) {
	svcPorts := "tcp,1883,1883|default.mosquitto-host-edge1"
	port := 1883
	protoName, svcName := getProtocol(svcPorts, port)
	fmt.Printf("protoName:%s, svcName:%s\n", protoName, svcName)
}
