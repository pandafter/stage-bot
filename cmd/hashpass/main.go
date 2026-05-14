package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/kart-academy/instagram-bot/internal/auth"
)

func main() {
	pwd := flag.String("pwd", "", "password to hash")
	flag.Parse()
	if *pwd == "" {
		fmt.Fprintln(os.Stderr, "usage: hashpass -pwd <password>")
		os.Exit(1)
	}
	hash, err := auth.Hash(*pwd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(hash)
}
