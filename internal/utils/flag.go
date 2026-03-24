package utils

import (
	"fmt"
	"net"
	"strings"
)

type IPv6List []net.IPNet

func (i *IPv6List) String() string {
	if i == nil || len(*i) == 0 {
		return ""
	}

	s := make([]string, len(*i))
	for i, ip := range *i {
		s[i] = ip.String()
	}
	return strings.Join(s, ",")
}

func (i *IPv6List) Set(value string) error {
	for _, part := range strings.Split(value, ",") {
		_, ip, err := net.ParseCIDR(strings.TrimSpace(part))
		if err != nil {
			return err
		}

		if ip.IP.To4() != nil {
			return fmt.Errorf("not an IPv6 address: %s", part)
		}

		*i = append(*i, *ip)
	}

	return nil
}
