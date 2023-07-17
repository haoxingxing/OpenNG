package auth

import (
	"net"
	"sync"

	"github.com/dlclark/regexp2"
	"github.com/haoxingxing/OpenNG/http"
	"github.com/haoxingxing/OpenNG/tcp"
)

type knockauthMgr struct {
	whitelist sync.Map
}

func (mgr *knockauthMgr) Handle(c *tcp.Connection) tcp.SerRet {
	host, _, _ := net.SplitHostPort(c.Addr().String())
	esx, ok := mgr.whitelist.Load(host)
	if ok && esx.(bool) {
		return tcp.Continue
	} else {
		return tcp.Close
	}
}
func (mgr *knockauthMgr) HandleHTTP(ctx *http.HttpCtx) http.Ret {
	host, _, _ := net.SplitHostPort(ctx.Req.RemoteAddr)
	if ctx.Req.URL.Path != "/" {
		host = ctx.Req.URL.Path[1:]
	}
	if _, ok := mgr.whitelist.Load(host); ok {
		ctx.Resp.WriteHeader(http.StatusOK)
		ctx.WriteString("DOOR OPENED ALREADY\n" + host)
	} else {
		mgr.whitelist.Store(host, true)
		ctx.WriteString("DOOR OPEN\n" + host)
	}
	return http.RequestEnd
}

func (mgr *knockauthMgr) Hosts() []*regexp2.Regexp {
	return nil
}
func NewKnockMgr() *knockauthMgr {
	mgr := &knockauthMgr{}
	return mgr
}
