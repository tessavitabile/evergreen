package agent

import (
	"bytes"
	"fmt"
	"github.com/evergreen-ci/evergreen"
	"io"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"
)

// DumpStackOnSIGQUIT listens for a SIGQUIT signal and writes stack dump to the
// given io.Writer when one is received. Blocks, so spawn it as a goroutine.
func DumpStackOnSIGQUIT(curAgent **Agent) {
	in := make(chan os.Signal)
	signal.Notify(in, syscall.SIGQUIT)
	for range in {
		var buf [32 * 1024]byte
		task := "no running task"
		command := "no running command"
		n := runtime.Stack(buf[:], true)
		stackBytes := buf[:n]
		agt := *curAgent
		if agt != nil {
			if agt.taskConfig != nil {
				task = agt.taskConfig.Task.Id
			}
			if cmd := agt.CurrentCommand(); cmd != nil {
				command = cmd.Command
			}
		}

		// we dump to files and logs without blocking, in case our logging is deadlocked or broken
		go dumpToDisk(task, command, stackBytes)
		go dumpToLogs(task, command, stackBytes, agt)

	}
}

func dumpToLogs(task, command string, stack []byte, agt *Agent) {
	if agt != nil && agt.logger != nil {
		logWriter := evergreen.NewInfoLoggingWriter(agt.logger.System)
		dumpDebugInfo(task, command, stack, logWriter)
	}
}

func dumpToDisk(task, command string, stack []byte) {
	dumpFile, err := os.Create(newDumpFilename())
	if err != nil {
		return // fail silently -- things are very wrong
	}
	defer dumpFile.Close()
	dumpDebugInfo(task, command, stack, dumpFile)
	dumpFile.Close()
}

func newDumpFilename() string {
	return fmt.Sprintf("evergreen_agent_%_dump_%v.log", os.Getpid(), time.Now().Format(time.RFC3339))
}

func dumpDebugInfo(task, command string, stack []byte, w io.Writer) {
	out := bytes.Buffer{}
	out.WriteString(fmt.Sprintf("Agent dump taken at %v.\n\n", time.Now().Format(time.UnixDate)))
	out.WriteString(fmt.Sprintf("Running command '%v' for task '%v'.\n\n", command, task))
	out.Write(stack)
	w.Write(out.Bytes())
}
