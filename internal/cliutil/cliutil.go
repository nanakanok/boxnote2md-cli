// Package cliutil は flag.FlagSet 周りのユーティリティ。
//
// Go 標準の flag パッケージはフラグを位置引数より前に書く必要があるが、
// CLI として `cmd <input> -o ./out` の順を許容したいので、引数を
// 「フラグ + 値」と「位置引数」に分離して並べ替える。
package cliutil

import (
	"flag"
	"strings"
)

// ReorderArgs は args を「フラグ群 + 位置引数群」の順に並べ替える。
// fs は事前に FlagVar 登録を済ませた状態である必要がある (bool 判定に使う)。
func ReorderArgs(fs *flag.FlagSet, args []string) []string {
	bools := boolFlagNames(fs)

	var flags, positional []string
	expectValue := false
	for _, a := range args {
		if expectValue {
			flags = append(flags, a)
			expectValue = false
			continue
		}
		if strings.HasPrefix(a, "-") && a != "-" && a != "--" {
			flags = append(flags, a)
			// `-name=value` は値まで含むので追加値は不要
			if strings.Contains(a, "=") {
				continue
			}
			name := strings.TrimLeft(a, "-")
			if !bools[name] {
				expectValue = true
			}
			continue
		}
		positional = append(positional, a)
	}
	return append(flags, positional...)
}

func boolFlagNames(fs *flag.FlagSet) map[string]bool {
	out := map[string]bool{}
	fs.VisitAll(func(f *flag.Flag) {
		if g, ok := f.Value.(interface{ IsBoolFlag() bool }); ok && g.IsBoolFlag() {
			out[f.Name] = true
		}
	})
	return out
}
