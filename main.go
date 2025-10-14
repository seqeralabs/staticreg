package main

import (
	"github.com/seqeralabs/staticreg/cmd"
	"github.com/seqeralabs/staticreg/pkg/db"
)

func main() {
	db.InitPool()
	defer db.ClosePool()
	cmd.Execute()

}
