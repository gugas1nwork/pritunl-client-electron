package connection

import (
	"fmt"
	"net"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dropbox/godropbox/container/set"
	"github.com/pritunl/pritunl-client-electron/service/utils"
)

var (
	ipReg       = regexp.MustCompile(`(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)(\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)){3}`)
	profileReg  = regexp.MustCompile(`[^a-z0-9_\- ]+`)
	restartLock sync.Mutex
)

var safeChars = set.NewSet(
	'a',
	'b',
	'c',
	'd',
	'e',
	'f',
	'g',
	'h',
	'i',
	'j',
	'k',
	'l',
	'm',
	'n',
	'o',
	'p',
	'q',
	'r',
	's',
	't',
	'u',
	'v',
	'w',
	'x',
	'y',
	'z',
	'A',
	'B',
	'C',
	'D',
	'E',
	'F',
	'G',
	'H',
	'I',
	'J',
	'K',
	'L',
	'M',
	'N',
	'O',
	'P',
	'Q',
	'R',
	'S',
	'T',
	'U',
	'V',
	'W',
	'X',
	'Y',
	'Z',
	'0',
	'1',
	'2',
	'3',
	'4',
	'5',
	'6',
	'7',
	'8',
	'9',
	'-',
	'.',
	':',
	'[',
	']',
)

func FilterHostStr(s string, n int) string {
	if len(s) == 0 {
		return ""
	}

	if len(s) > n {
		s = s[:n]
	}

	ns := ""
	for _, c := range s {
		if safeChars.Contains(c) {
			ns += string(c)
		}
	}

	return ns
}

func ParseAddress(input string) (addr string) {
	input = FilterHostStr(input, 256)

	endBracketIndex := strings.LastIndex(input, "]")
	if strings.HasPrefix(input, "[") && endBracketIndex != -1 {
		addr = input[1:endBracketIndex]
		if strings.Contains(addr, ":") {
			ip := net.ParseIP(addr)
			if ip != nil {
				addr = "[" + ip.String() + "]"
			}
		}

		colonIndex := strings.LastIndex(input, ":")
		if colonIndex > endBracketIndex {
			port, _ := strconv.Atoi(input[colonIndex+1:])
			if port != 0 && port != 443 {
				addr += fmt.Sprintf(":%d", port)
			}
		}

		return
	}

	if strings.Contains(input, ":") {
		ip := net.ParseIP(input)
		if ip != nil {
			addr = "[" + ip.String() + "]"
			return
		}

		colonIndex := strings.LastIndex(input, ":")
		addr = input[:colonIndex]
		if strings.Contains(addr, ":") {
			ip := net.ParseIP(addr)
			if ip != nil {
				addr = "[" + ip.String() + "]"
			}
		}

		port, _ := strconv.Atoi(input[colonIndex+1:])
		if port != 0 && port != 443 {
			addr += fmt.Sprintf(":%d", port)
		}

		return
	}

	addr = input
	return
}

func RestartProfiles() (err error) {
	restartLock.Lock()
	defer restartLock.Unlock()

	conns := GlobalStore.GetAll()

	for _, conn := range conns {
		conn.StopBackground()
	}

	for _, conn := range conns {
		conn.StopWait()
	}

	// TOOD Iter and restart

	return
}

func Clean() (err error) {
	if runtime.GOOS != "windows" {
		return
	}

	for i := 0; i < 10; i++ {
		_, _ = utils.ExecCombinedOutput(
			"sc.exe", "stop", fmt.Sprintf("WireGuardTunnel$pritunl%d", i),
		)
		time.Sleep(100 * time.Millisecond)
		_, _ = utils.ExecCombinedOutput(
			"sc.exe", "delete", fmt.Sprintf("WireGuardTunnel$pritunl%d", i),
		)
	}

	return
}