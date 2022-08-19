package mdns

import (
	"bytes"
	"crypto/md5"
	"encoding/binary"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	relayv1 "github.com/libp2p/go-libp2p/p2p/protocol/circuitv1/relay"
	"io"
	mrand "math/rand"
	"os"
)

func HostnameOrDie() string {
	hostname, err := os.Hostname()
	if err != nil {
		panic(err)
	}
	return hostname
}

func GenerateKeyPairWithString(s string) (crypto.PrivKey, error) {
	m := md5.New()
	_, err := io.WriteString(m, s)
	if err != nil {
		return nil, err
	}
	reader := bytes.NewReader(m.Sum(nil))

	var seed int64
	err = binary.Read(reader, binary.BigEndian, &seed)
	if err != nil {
		return nil, err
	}
	r := mrand.New(mrand.NewSource(seed))
	privKey, _, err := crypto.GenerateKeyPairWithReader(crypto.Ed25519, -1, r)
	if err != nil {
		return privKey, err
	}
	return privKey, nil
}

//判断当前节点是否启动relay中继
func EnableRelayorNot(host host.Host) {

	// 如果拥有公网地址，启动作为中继
	if HostnameOrDie() == "k8s-master" || HostnameOrDie() == "ke-edge1" {
		// TODO
		r, err := relayv1.NewRelay(host)
		if err != nil {
			panic(err)
		}
		defer r.Close()
	}

	//如果设置当前节点作为中继，启动作为中继

}
