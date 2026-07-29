// Command fixturemeta-digest prints fixture digests for the generator script.
package main

import (
	"fmt"
	"os"

	"github.com/mewisme/mew/tools/conformance/fixturemeta"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "usage: %s <source-tree|file> <path>\n", os.Args[0])
		os.Exit(2)
	}
	switch os.Args[1] {
	case "source-tree":
		digest, err := fixturemeta.SourceTreeDigest(os.Args[2])
		if err != nil {
			fmt.Fprintf(os.Stderr, "source-tree: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(digest)
	case "file":
		hash, err := fixturemeta.FileSHA256(os.Args[2])
		if err != nil {
			fmt.Fprintf(os.Stderr, "file: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(hash)
	default:
		fmt.Fprintf(os.Stderr, "unknown mode %q\n", os.Args[1])
		os.Exit(2)
	}
}
