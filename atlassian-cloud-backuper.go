package main

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2024 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	_ "embed"

	APP "github.com/essentialkaos/atlassian-cloud-backuper/app"
)

// ////////////////////////////////////////////////////////////////////////////////// //

//go:embed go.mod
var gomod []byte

// gitrev is short hash of the latest git commit
var gitrev string

// ////////////////////////////////////////////////////////////////////////////////// //

func main() {
	APP.Run(gitrev, gomod)
}
