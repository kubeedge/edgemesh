#!/bin/sh

# check if edgecore is installed
if pgrep "edgecore" >/dev/null; then
  echo "Edgecore process is running. Continuing with the initialization steps."

  # copy cni to host
	rm -f /opt/cni/bin/egdemesh-cni.old || true
	( [ -f "/opt/cni/bin/egdemesh-cni" ] && mv /usr/local/bin/edgemesh-cni /opt/cni/bin/egdemesh-cni.old ) || true
	cp /usr/local/bin/edgemesh-cni /opt/cni/bin/egdemesh-cni
	rm -f /opt/cni/bin/egdemesh-cni.old &>/dev/null  || true

  # generate 10-edgemesh-cni.conflist under /etc/edgemesh/config/
  cat <<EOF > /etc/edgemesh/config/10-edgemesh-cni.conflist
{
  "cniVersion": "0.0.1",
  "name": "edgemesh",
  "type": "edgemesh",
  "delegate": {
    "cniVersion": "0.0.1",
    "type":"bridge",
    "ipam": {
      "type":"spiderpool"
    }
  }
}
EOF

  # cpoy it to  /etc/cni/net.d
  cp -f /etc/edgemesh/config/10-edgemesh-cni.conflist /etc/cni/net.d/

else
  echo "Edgecore process is not running. Exiting."
  exit 1
fi