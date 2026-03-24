package utils

import (
	"testing"
)

func TestIPv6ListString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		list *IPv6List
		want string
	}{
		{
			name: "nil list",
			list: nil,
			want: "",
		},
		{
			name: "empty list",
			list: func() *IPv6List {
				list := IPv6List{}
				return &list
			}(),
			want: "",
		},
		{
			name: "joined networks",
			list: func() *IPv6List {
				list := IPv6List{}
				if err := list.Set("2001:db8::/32,2001:db8:1::/48"); err != nil {
					t.Fatalf("Set() error = %v", err)
				}
				return &list
			}(),
			want: "2001:db8::/32,2001:db8:1::/48",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := test.list.String()
			if got != test.want {
				t.Fatalf("String() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestIPv6ListSet(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		initial IPv6List
		value   string
		want    string
		wantErr bool
	}{
		{
			name:  "parses single network",
			value: "2001:db8::/32",
			want:  "2001:db8::/32",
		},
		{
			name:  "parses multiple networks with spaces",
			value: "2001:db8::/32, 2001:db8:1::/48",
			want:  "2001:db8::/32,2001:db8:1::/48",
		},
		{
			name: "appends to existing list",
			initial: func() IPv6List {
				list := IPv6List{}
				if err := list.Set("2001:db8::/32"); err != nil {
					t.Fatalf("Set() error = %v", err)
				}
				return list
			}(),
			value: "2001:db8:1::/48",
			want:  "2001:db8::/32,2001:db8:1::/48",
		},
		{
			name:    "rejects ipv4 network",
			value:   "192.0.2.0/24",
			wantErr: true,
		},
		{
			name:    "rejects invalid cidr",
			value:   "not-a-cidr",
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			list := append(IPv6List(nil), test.initial...)
			err := list.Set(test.value)
			if test.wantErr {
				if err == nil {
					t.Fatal("Set() error = nil, want error")
				}
				return
			}

			if err != nil {
				t.Fatalf("Set() error = %v", err)
			}
			if got := list.String(); got != test.want {
				t.Fatalf("String() after Set() = %q, want %q", got, test.want)
			}
		})
	}
}
