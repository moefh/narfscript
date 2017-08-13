package main

import (
	"fmt"
	"os"
        "./narfscript"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("USAGE: %s filename\n", os.Args[0])
		return
	}
	filename := os.Args[1]
	args := make([]narfscript.Value, 0, len(os.Args)-2)
	for _, arg := range os.Args[2:] {
		args = append(args, narfscript.NewValueString(arg))
	}

	narf := narfscript.NewNarf()
	if err := narf.Parse(filename); err != nil {
		fmt.Printf("%s\n", err)
		return
	} else {
		narf.DumpEnv()
		narf.DumpFunctions()
	}

	_, err := narf.CallFunction("main", args)
	if err != nil {
		fmt.Printf("%s\n", err)
	}
}
