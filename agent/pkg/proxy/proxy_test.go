package proxy

import (
	"testing"
)

func TestGetProtocol(t *testing.T) {
	tests := []struct {
		name          string
		svcPorts      string
		port          int
		wantProtoName string
		wantSvcName   string
	}{
		{
			"protocol parse",
			"mqtt,1883|svc-mosquitto",
			1883,
			"mqtt",
			"svc-mosquitto",
		},
		{
			"Endpoints -> Service",
			"tcp,1883,1883|default.mosquitto-host-edge1",
			1883,
			"tcp",
			"default.mosquitto-host-edge1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			protoName, svcName := getProtocol(tt.svcPorts, tt.port)
			if protoName == "" || svcName == "" {
				t.Errorf("getProtocol() failed, svcPorts = %v, port %v", tt.svcPorts, tt.port)
				return
			}

			if protoName != tt.wantProtoName && svcName != tt.wantSvcName {
				t.Errorf("getProtocol() = (%v, %v), want (%v, %v)", protoName, svcName, tt.wantProtoName, tt.wantSvcName)
			}
		})
	}
}
