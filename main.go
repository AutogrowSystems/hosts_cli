package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/exec"
	"strings"
)

type line struct {
	ip, hostname string
}

const filename = "/etc/hosts"

func containsPart(haystack, needle string) bool {
	return strings.Contains(haystack, "\t"+needle) || strings.Contains(haystack, needle+"\t")
}


func reverse(a []int) []int {
	for i := len(a)/2 - 1; i >= 0; i-- {
		opp := len(a) - 1 - i
		a[i], a[opp] = a[opp], a[i]
	}

	return a
}

func amIRoot() bool {
	cmd := exec.Command("whoami")
	user, err := cmd.Output()
	if err != nil {
		// couldn't determine root due to error, run anyway - the user won't be able
		// to mod anything without root rights anyway
		return true
	}

	return strings.TrimSpace(string(user)) == "root"
}

type hostlist struct {
	lines   []string
	changed bool
}

func (hl *hostlist) Read(fn string) error {
	b, err := ioutil.ReadFile(fn)
	if err != nil {
		return err
	}

	hl.Parse(b)
	return nil
}

func (hl *hostlist) Parse(b []byte) {
	hl.lines = strings.Split(string(b), "\n")
}

func (hl *hostlist) Write(fn string) error {
	return ioutil.WriteFile(fn, hl.Bytes(), 0644)
}

func (hl *hostlist) Bytes() []byte {
	return []byte(strings.Join(hl.lines, "\n"))

func (hl *hostlist) Contains(a, b string) (bool, error) {
	var ip, hostname string

	if net.ParseIP(a) == nil && net.ParseIP(b) == nil {
		return false, fmt.Errorf("neither %s or %s is a valid IP address", a, b)
	}

	if net.ParseIP(a) == nil {
		hostname = a
		ip = b
	}

	for _, line := range hl.lines {
		if line == fmt.Sprintf("%s\t%s", ip, hostname) {
			return true, nil
		}
	}

	return false, nil
}

func (hl *hostlist) Add(a, b string) error {
	var ip, hostname string

	if net.ParseIP(a) == nil && net.ParseIP(b) == nil {
		return fmt.Errorf("neither %s or %s is a valid IP address", a, b)
	}

	if net.ParseIP(a) == nil {
		hostname = a
		ip = b
	}

	hl.lines = append(hl.lines, fmt.Sprintf("%s\t%s", ip, hostname))
	hl.changed = true
	return nil
}

func (hl *hostlist) Remove(thing string) error {
	deletes := []int{}
	for i, line := range hl.lines {
		if containsPart(line, thing) {
			deletes = append(deletes, i)
		}
	}

	for _, i := range reverse(deletes) {
		hl.lines = append(hl.lines[:i], hl.lines[i+1:]...)
	}

	hl.changed = true
	return nil
}

func (hl *hostlist) Comment(thing string) error {
	for i, line := range hl.lines {
		if containsPart(line, thing) {
			hl.lines[i] = "#" + line
		}
	}

	hl.changed = true
	return nil
}

func (hl *hostlist) Uncomment(thing string) error {
	for i, line := range hl.lines {
		if containsPart(line, thing) {
			hl.lines[i] = strings.TrimLeft(line, "#")
		}
	}

	hl.changed = true
	return nil
}

func main() {
	log.SetFlags(0)

	if !amIRoot() {
		log.Fatal("Please run this program as Root!")
	}

	if len(os.Args) <= 1 {
		log.Fatal("Nothing to do, please specify command")
	}

	hosts := hostlist{}
	hosts.Read(filename)

	command := string(os.Args[1])

	// fmt.Println(command)
	switch command {
	default:
		log.Fatalf("Unknown command: %s", command)

	case "list", "ls":
		fmt.Println(string(hosts.Bytes()))

	case "del", "rm", "-":
		if len(os.Args) != 3 {
			log.Fatal("Please give an IP or hostname to delete")
		}
		hosts.Remove(os.Args[2])

	case "ucom":
		if len(os.Args) != 3 {
			log.Fatal("Please give an IP or hostname to uncomment")
		}
		hosts.Uncomment(os.Args[2])

	case "com":
		if len(os.Args) != 3 {
			log.Fatal("Please give an IP or hostname to comment out")
		}
		hosts.Comment(os.Args[2])

	case "add", "+":
		if len(os.Args) != 4 {
			log.Fatal("Please give arguments in the form ip, hostname")
		}
		err := hosts.Add(os.Args[2], os.Args[3])
		if err != nil {
			log.Fatal(err)
		}

	case "has", "?", "contains":
		if len(os.Args) != 4 {
			log.Fatal("Please give arguments in the form ip, hostname")
		}

		yes, err := hosts.Contains(os.Args[2], os.Args[3])
		if err != nil {
			log.Fatal(err)
		}

		if yes {
			os.Exit(0) // exit code 0 means it was contained within
		}

		os.Exit(1) // exit code 1 means not contained within

	}

	if hosts.changed {
		log.Printf("writing changes to %s", filename)
		hosts.Write(filename)
	}
}
