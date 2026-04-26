package main

import "os/exec"

func openInBrowser(url string) {
	exec.Command("open", url).Start()
}
