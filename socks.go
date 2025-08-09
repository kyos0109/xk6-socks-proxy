package proxy

import (
	"github.com/kyos0109/xk6-socks-proxy/pkg/proxy"
	"go.k6.io/k6/js/modules"
)

func init() {
	modules.Register("k6/x/xk6-socks-proxy", proxy.New())
}
