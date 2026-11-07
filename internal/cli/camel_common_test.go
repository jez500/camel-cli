// Copyright 2026 Jeremy Graham and contributors. Licensed under Apache-2.0. See LICENSE.

package cli

import "testing"

func TestMoneyFormatting(t *testing.T) {
	v := func(f float64) *float64 { return &f }
	cases := []struct {
		p      *float64
		locale string
		want   string
	}{
		{v(3600.99), "us", "$3,600.99"},
		{v(19.98), "au", "$19.98"},
		{v(1234567.5), "uk", "£1,234,567.50"},
		{v(9.99), "de", "€9.99"},
		{nil, "us", "-"},
	}
	for _, c := range cases {
		if got := money(c.p, c.locale); got != c.want {
			t.Errorf("money(%v, %s) = %q, want %q", c.p, c.locale, got, c.want)
		}
	}
}
