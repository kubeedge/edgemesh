package protocol

var RegisterProtocols []string

type Protocol interface {
	Process()
	Register()
}
