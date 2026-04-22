package main
import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)
func withinRoot(p, root string) bool {
	rel, err := filepath.Rel(root, p)
	if err != nil { return false }
	if rel == "." { return true }
	return !strings.HasPrefix(rel, "..")
}
func main() {
	wd, _ := os.Getwd()
	root := filepath.Join(wd, "root")
	linkPath := filepath.Join(root, "link")
	
	fmt.Printf("Root: %q\n", root)
	fmt.Printf("Link path (input): %q\n", linkPath)
	
	evaluated, err := filepath.EvalSymlinks(linkPath)
	if err != nil {
		fmt.Printf("EvalSymlinks error: %v\n", err)
	} else {
		fmt.Printf("EvalSymlinks result: %q\n", evaluated)
		fmt.Printf("withinRoot: %v\n", withinRoot(evaluated, root))
	}
}
