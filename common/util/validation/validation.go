package validation

var supportedTransports = []string{"tcp", "ws", "quic"}

func IsValidTransport(transport string) []string {
	for _, tr := range supportedTransports {
		if transport == tr {
			return nil
		}
	}
	return supportedTransports
}
