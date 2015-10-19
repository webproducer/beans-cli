package main

import (
	"flag"
	"fmt"
	"github.com/kr/beanstalk"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

func main() {
	commandArgs, other := parseCommands()
	beansAddr := parseArgs(other)
	c, err := beanstalk.Dial("tcp", beansAddr)
	if err != nil {
		fmt.Printf("Warning: can't connect to beanstalk on %s (use -b arg to specify correct connection addr:port)\n", beansAddr)
	} else {
		defer c.Close()
	}
	if len(commandArgs) == 0 {
		printUsageInfo()
		os.Exit(1)
	}
	switch commandArgs[0] {
	case "help":
		helpCmd(c)
		os.Exit(0)
	case "server-info":
		checkConn(c)
		s, _ := c.Stats()
		fmt.Println("Global server info:\n")
		printStats(s)
	default:
		checkConn(c)
		tubeCmd(c, commandArgs[0], commandArgs[1:])
	}
}

func checkConn(c *beanstalk.Conn) {
	if c == nil {
		fmt.Println("You're not connected to beanstalkd")
		os.Exit(2)
	}
}

func printStats(s map[string]string) {
	maxLen := 0
	keys := make([]string, 0, len(s))
	for k, _ := range s {
		if len(k) > maxLen {
			maxLen = len(k)
		}
		keys = append(keys, k)
	}
	sort.Sort(sort.StringSlice(keys))
	for _, k := range keys {
		offset := fmt.Sprintf("%d", maxLen+1)
		fmt.Printf("%-"+offset+"s: %s\n", k, s[k])
	}
}

func helpCmd(c *beanstalk.Conn) {
	printUsageInfo()
	fmt.Println("\nAvailable tube actions:")
	fmt.Println("- info")
	fmt.Println("- kick [bound]")
	fmt.Println("- delete all|{id}")
	fmt.Println("- put {data} [pri] [delay] [ttr]")
	if c != nil {
		tubes, _ := c.ListTubes()
		fmt.Printf("\nAvailable tubes: %s\n", strings.Join(tubes, " | "))
	}
	fmt.Println("\nGlobal args:")
	fmt.Println("-b=127.0.0.1:11300\tBeanstalkd [addr]:port")
	fmt.Println("")
}

func tubeCmd(c *beanstalk.Conn, tubeName string, args []string) {
	tube := &beanstalk.Tube{c, tubeName}
	if len(args) == 0 || args[0] == "info" {
		fmt.Printf("Info for tube %s:\n\n", tubeName)
		s, _ := tube.Stats()
		printStats(s)
		return
	}
	cmd := args[0]
	actionArgs := args[1:]
	//TODO: implement job-stats, pause
	switch cmd {
	case "kick":
		kickTube(tube, actionArgs)
	case "delete":
		deleteFromTube(tube, actionArgs)
	case "put":
		putToTube(tube, actionArgs)
	default:
		fmt.Printf("Unknown command '%s'. Type 'beans-cli help' to see usage information\n", cmd)
		os.Exit(3)
	}
}

func kickTube(t *beanstalk.Tube, args []string) {
	cnt := 100000
	if len(args) > 0 {
		if c, err := strconv.Atoi(args[0]); err != nil {
			fmt.Printf("Wrong argument for kick: %s\n", args[0])
			os.Exit(3)
		} else {
			cnt = c
		}
	}
	n, err := t.Kick(cnt)
	if err != nil {
		fmt.Println(err)
		os.Exit(4)
	}
	fmt.Println(n, "items kicked")
}

func deleteFromTube(t *beanstalk.Tube, args []string) {
	if len(args) == 0 {
		fmt.Println("You must specify argument for delete: all|{job_id}")
		os.Exit(3)
	}
	if args[0] == "all" {
		deleteAllFromTube(t)
		return
	}
	id, err := strconv.ParseUint(args[0], 10, 64)
	if err != nil {
		fmt.Printf("Wrong argument for delete: %s\n", args[0])
		os.Exit(3)
	}
	if err := t.Conn.Delete(id); err != nil {
		fmt.Println(err)
		os.Exit(4)
	}
	fmt.Printf("Job %d deleted\n", id)
}

func deleteAllFromTube(t *beanstalk.Tube) {
	queue := beanstalk.NewTubeSet(t.Conn, t.Name)
	deleted := 0
	for {
		id, _, err := queue.Reserve(3 * time.Second)
		if err != nil {
			break
		}
		if err := t.Conn.Delete(id); err != nil {
			fmt.Println(err)
			continue
		}
		deleted++
	}
	fmt.Printf("%d jobs deleted from %s\n", deleted, t.Name)
}

func putToTube(t *beanstalk.Tube, args []string) {
	var data []byte
	var pri uint32
	var delay time.Duration
	ttr := 30 * time.Second
	switch len(args) {
	case 4:
		if ttrSecs, err := strconv.Atoi(args[3]); err == nil {
			ttr = time.Duration(ttrSecs) * time.Second
		}
		fallthrough
	case 3:
		if delaySecs, err := strconv.Atoi(args[2]); err == nil {
			delay = time.Duration(delaySecs) * time.Second
		}
		fallthrough
	case 2:
		if pri64, err := strconv.ParseUint(args[1], 10, 32); err == nil {
			pri = uint32(pri64)
		}
		fallthrough
	case 1:
		data = []byte(args[0])
	}
	id, err := t.Put(data, pri, delay, ttr)
	if err != nil {
		fmt.Println(err)
		os.Exit(4)
	}
	fmt.Printf("Job was put to tube %s with pri=%d, delay=%v, ttr=%v. Job ID: %d\n", t.Name, pri, delay, ttr, id)
}

func parseCommands() (commands, otherArgs []string) {
	commands = make([]string, 0, len(os.Args)-1)
	for _, cmd := range os.Args[1:] {
		if strings.HasPrefix(cmd, "-") {
			break
		}
		commands = append(commands, cmd)
	}
	otherArgs = os.Args[len(commands)+1:]
	return commands, otherArgs
}

func parseArgs(args []string) (addr string) {
	f := flag.NewFlagSet("global", flag.ExitOnError)
	beanstalkAddrFlag := f.String("b", "127.0.0.1:11300", "Beanstalkd [addr]:port")
	f.Parse(args)
	return *beanstalkAddrFlag
}

func printUsageInfo() {
	fmt.Println("\nBase usage: beans-cli (help | server-info | {tube_name} [tube_action]) [{global_args}]")
}
